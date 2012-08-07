// Copyright ©2012 Dan Kortschak <dan.kortschak@adelaide.edu.au>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package interval implements an interval tree based on an augmented
// Left-Leaning Red Black tree.
package interval

import (
	"errors"
)

const (
	TD234 = iota
	BU23
)

// Operation mode of the underlying LLRB tree.
const Mode = BU23

func init() {
	if Mode != TD234 && Mode != BU23 {
		panic("interval: unknown mode")
	}
}

// ErrInvertedRange is returned if an Overlapper, o is used where the minimum values is
// greater than the maximum value according to Compare().
var ErrInvertedRange = errors.New("interval: inverted range")

// An Overlapper is a type that can be inserted into a Tree or used as a range
// or equality query on the tree,
type Overlapper interface {
	// Overlap returns a value indicating the sort order relationship between the
	// receiver and the parameter.
	//
	// Given c = a.Overlap(b):
	//  c < 0 if a.Max() ≲ b.Min();
	//  c == 0 if a.Max() ≳ b.Min() && a.Min() ≲ b.Max(); and
	//  c > 0 if a.Min() ≳ b.Max().
	//
	// The details of inequalities between a and b can be used to specify overlap/match
	// requirements and indexing types, for example half-open or closed.
	Overlap(Overlapper) int
	// Return a Comparable equal to the Minimum value of the Overlapper.
	Min() Comparable
	// Return a Comparable equal to the Maximum value of the Overlapper.
	Max() Comparable
	// Returns an mutable copy of the Overlapper.
	Mutable() Mutable
}

// A Mutable is an Overlapper that can have its range altered.
type Mutable interface {
	Overlapper
	SetMin(Comparable) // Set the minimum value.
	SetMax(Comparable) // Set the maximum value.
}

// A Comparable is a type that describes the ends of an Overlapper.
type Comparable interface {
	// Compare returns a value indicating the sort order relationship between the
	// receiver and the parameter.
	//
	// Given c = a.Compare(b):
	//  c < 0 if a < b;
	//  c == 0 if a == b; and
	//  c > 0 if a > b.
	//
	Compare(Comparable) int
}

// A Color represents the color of a Node.
type Color bool

// String returns a string representation of a Color.
func (c Color) String() string {
	if c {
		return "Black"
	}
	return "Red"
}

const (
	// Red as false give us the defined behaviour that new nodes are red. Although this
	// is incorrect for the root node, that is resolved on the first insertion.
	Red   Color = false
	Black Color = true
)

// A Node represents a node in the LLRB tree.
type Node struct {
	Elem        Overlapper
	Range       Mutable
	Left, Right *Node
	Color       Color
}

// A Tree manages the root node of an interval tree. Public methods are exposed through this type.
type Tree struct {
	Root  *Node // Root node of the tree.
	Count int   // Number of elements stored.
}

// Helper methods

// color returns the effect color of a Node. A nil node returns black.
func (self *Node) color() Color {
	if self == nil {
		return Black
	}
	return self.Color
}

// (a,c)b -rotL-> ((a,)b,)c
func (self *Node) rotateLeft() (root *Node) {
	// Assumes: self has two children.
	root = self.Right
	if root.Left != nil {
		self.Range.SetMax(root.Left.Range.Max())
	} else {
		self.Range.SetMax(self.Elem.Max())
	}
	root.Range.SetMin(self.Range.Min())
	self.Right = root.Left
	root.Left = self
	root.Color = self.Color
	self.Color = Red
	return
}

// (a,c)b -rotR-> (,(,c)b)a
func (self *Node) rotateRight() (root *Node) {
	// Assumes: self has two children.
	root = self.Left
	if root.Right != nil {
		self.Range.SetMin(root.Right.Range.Min())
	} else {
		self.Range.SetMin(self.Elem.Min())
	}
	root.Range.SetMax(self.Range.Max())
	self.Left = root.Right
	root.Right = self
	root.Color = self.Color
	self.Color = Red
	return
}

// (aR,cR)bB -flipC-> (aB,cB)bR | (aB,cB)bR -flipC-> (aR,cR)bB 
func (self *Node) flipColors() {
	// Assumes: self has two children.
	self.Color = !self.Color
	self.Left.Color = !self.Left.Color
	self.Right.Color = !self.Right.Color
}

// fixUp ensures that black link balance is correct, that red nodes lean left,
// and that 4 nodes are split in the case of BU23 and properly balanced in TD234.
func (self *Node) fixUp() *Node {
	if self.Right.color() == Red {
		if Mode == TD234 && self.Right.Left.color() == Red {
			self.Right = self.Right.rotateRight()
		}
		self = self.rotateLeft()
	}
	if self.Left.color() == Red && self.Left.Left.color() == Red {
		self = self.rotateRight()
	}
	if Mode == BU23 && self.Left.color() == Red && self.Right.color() == Red {
		self.flipColors()
	}
	return self
}

func (self *Node) moveRedLeft() *Node {
	self.flipColors()
	if self.Right.Left.color() == Red {
		self.Right = self.Right.rotateRight()
		self = self.rotateLeft()
		self.flipColors()
		if Mode == TD234 && self.Right.Right.color() == Red {
			self.Right = self.Right.rotateLeft()
		}
	}
	return self
}

func (self *Node) moveRedRight() *Node {
	self.flipColors()
	if self.Left.Left.color() == Red {
		self = self.rotateRight()
		self.flipColors()
	}
	return self
}

// Len returns the number of intervals stored in the Tree.
func (self *Tree) Len() int {
	return self.Count
}

// Get returns the a slice of Overlappers that overlap q in the Tree according
// to Overlap.
func (self *Tree) Get(q Overlapper) (o []Overlapper, err error) {
	if q.Min().Compare(q.Max()) > 0 {
		return nil, ErrInvertedRange
	}
	if self.Root == nil {
		return
	}
	self.Root.doMatch(
		func(e Overlapper) (done bool) {
			o = append(o, e)
			return
		},
		q,
	)

	return
}

// Insert inserts the Overlapper e into the Tree. Insertions do not replace
// existing stored intervals.
func (self *Tree) Insert(e Overlapper) (err error) {
	if e.Min().Compare(e.Max()) > 0 {
		return ErrInvertedRange
	}
	var d int
	self.Root, d = self.Root.insert(e)
	self.Count += d
	self.Root.Color = Black
	return
}

func (self *Node) insert(e Overlapper) (root *Node, d int) {
	if self == nil {
		return &Node{Elem: e, Range: e.Mutable()}, 1
	} else if self.Elem == nil {
		self.Elem = e
		self.Range.SetMin(e.Min())
		self.Range.SetMax(e.Max())
		return self, 1
	}

	if Mode == TD234 {
		if self.Left.color() == Red && self.Right.color() == Red {
			self.flipColors()
		}
	}

	switch c := e.Min().Compare(self.Elem.Min()); {
	case c < 0:
		self.Left, d = self.Left.insert(e)
	default:
		self.Right, d = self.Right.insert(e)
	}

	if self.Right.color() == Red && self.Left.color() == Black {
		self = self.rotateLeft()
	}
	if self.Left.color() == Red && self.Left.Left.color() == Red {
		self = self.rotateRight()
	}

	if Mode == BU23 {
		if self.Left.color() == Red && self.Right.color() == Red {
			self.flipColors()
		}
	}

	self.adjustRange()
	root = self

	return
}

func (self *Node) adjustRange() {
	if self.Left != nil {
		self.Range.SetMin(min(self.Range.Min(), self.Left.Range.Min()))
		self.Range.SetMax(max(self.Range.Max(), self.Left.Range.Max()))
	}
	if self.Right != nil {
		self.Range.SetMin(min(self.Range.Min(), self.Right.Range.Min()))
		self.Range.SetMax(max(self.Range.Max(), self.Right.Range.Max()))
	}
}

func min(a, b Comparable) Comparable {
	if a.Compare(b) < 0 {
		return a
	}
	return b
}

func max(a, b Comparable) Comparable {
	if a.Compare(b) > 0 {
		return a
	}
	return b
}

// DeleteMin deletes the left-most interval will be deleted.
func (self *Tree) DeleteMin() {
	if self.Root == nil {
		return
	}
	var d int
	self.Root, d = self.Root.deleteMin()
	self.Count += d
	if self.Root == nil {
		return
	}
	self.Root.Color = Black
}

func (self *Node) deleteMin() (root *Node, d int) {
	if self.Left == nil {
		return nil, -1
	}
	if self.Left.color() == Black && self.Left.Left.color() == Black {
		self = self.moveRedLeft()
	}
	self.Left, d = self.Left.deleteMin()

	root = self.fixUp()

	return
}

// DeleteMax deletes the right-most interval will be deleted.
func (self *Tree) DeleteMax() {
	if self.Root == nil {
		return
	}
	var d int
	self.Root, d = self.Root.deleteMax()
	self.Count += d
	if self.Root == nil {
		return
	}
	self.Root.Color = Black
}

func (self *Node) deleteMax() (root *Node, d int) {
	if self.Left != nil && self.Left.color() == Red {
		self = self.rotateRight()
	}
	if self.Right == nil {
		return nil, -1
	}
	if self.Right.color() == Black && self.Right.Left.color() == Black {
		self = self.moveRedRight()
	}
	self.Right, d = self.Right.deleteMax()

	root = self.fixUp()

	return
}

// Delete deletes the first node found that matches e according to Overlap().
func (self *Tree) Delete(e Overlapper) (err error) {
	if e.Min().Compare(e.Max()) > 0 {
		return ErrInvertedRange
	}
	if self.Root == nil {
		return
	}
	var d int
	self.Root, d = self.Root.delete(e)
	self.Count += d
	if self.Root == nil {
		return
	}
	self.Root.Color = Black
	return
}

func (self *Node) delete(e Overlapper) (root *Node, d int) {
	if e.Overlap(self.Elem) < 0 {
		if self.Left != nil {
			if self.Left.color() == Black && self.Left.Left.color() == Black {
				self = self.moveRedLeft()
			}
			self.Left, d = self.Left.delete(e)
		}
	} else {
		if self.Left.color() == Red {
			self = self.rotateRight()
		}
		if e.Overlap(self.Elem) == 0 && self.Right == nil {
			return nil, -1
		}
		if self.Right != nil {
			if self.Right.color() == Black && self.Right.Left.color() == Black {
				self = self.moveRedRight()
			}
			if e.Overlap(self.Elem) == 0 {
				self.Elem = self.Right.min().Elem
				self.Right, d = self.Right.deleteMin()
			} else {
				self.Right, d = self.Right.delete(e)
			}
		}
	}

	root = self.fixUp()

	return
}

// Return the left-most interval stored in the tree.
func (self *Tree) Min() Overlapper {
	if self.Root == nil {
		return nil
	}
	return self.Root.min().Elem
}

func (self *Node) min() (n *Node) {
	for n = self; n.Left != nil; n = n.Left {
	}
	return
}

// Return the right-most interval stored in the tree.
func (self *Tree) Max() Overlapper {
	if self.Root == nil {
		return nil
	}
	return self.Root.max().Elem
}

func (self *Node) max() (n *Node) {
	for n = self; n.Right != nil; n = n.Right {
	}
	return
}

// Floor returns the greatest interval equal to or less than the query q according to q.Overlap().
func (self *Tree) Floor(q Overlapper) (o Overlapper, err error) {
	if q.Min().Compare(q.Max()) > 0 {
		return nil, ErrInvertedRange
	}
	if self.Root == nil {
		return
	}
	n := self.Root.floor(q)
	if n == nil {
		return
	}
	return n.Elem, nil
}

func (self *Node) floor(q Overlapper) *Node {
	if self == nil {
		return nil
	}
	switch c := q.Overlap(self.Elem); {
	case c == 0:
		return self
	case c < 0:
		return self.Left.floor(q)
	default:
		if r := self.Right.floor(q); r != nil {
			return r
		}
	}
	return self
}

// Ceil returns the smallest value equal to or greater than the query q according to q.Overlap().
func (self *Tree) Ceil(q Overlapper) (o Overlapper, err error) {
	if q.Min().Compare(q.Max()) > 0 {
		return nil, ErrInvertedRange
	}
	if self.Root == nil {
		return
	}
	n := self.Root.ceil(q)
	if n == nil {
		return
	}
	return n.Elem, nil
}

func (self *Node) ceil(q Overlapper) *Node {
	if self == nil {
		return nil
	}
	switch c := q.Overlap(self.Elem); {
	case c == 0:
		return self
	case c > 0:
		return self.Right.ceil(q)
	default:
		if l := self.Left.ceil(q); l != nil {
			return l
		}
	}
	return self
}

// An Operation is a function that operates on an Overlapper. If done is returned true, the
// Operation is indicating that no further work needs to be done and so the Do function should
// traverse no further.
type Operation func(Overlapper) (done bool)

// Do performs fn on all intervals stored in the tree. A boolean is returned indicating whether the
// Do traversal was interrupted by an Operation returning true. If fn alters stored intervals' sort
// relationships, future tree operation behaviors are undefined.
func (self *Tree) Do(fn Operation) bool {
	if self.Root == nil {
		return false
	}
	return self.Root.do(fn)
}

func (self *Node) do(fn Operation) (done bool) {
	if self.Left != nil {
		done = self.Left.do(fn)
		if done {
			return
		}
	}
	done = fn(self.Elem)
	if done {
		return
	}
	if self.Right != nil {
		done = self.Right.do(fn)
	}
	return
}

// DoReverse performs fn on all intervals stored in the tree, but in reverse of sort order. A boolean
// is returned indicating whether the Do traversal was interrupted by an Operation returning true.
// If fn alters stored intervals' sort relationships, future tree operation behaviors are undefined.
func (self *Tree) DoReverse(fn Operation) bool {
	if self.Root == nil {
		return false
	}
	return self.Root.doReverse(fn)
}

func (self *Node) doReverse(fn Operation) (done bool) {
	if self.Right != nil {
		done = self.Right.doReverse(fn)
		if done {
			return
		}
	}
	done = fn(self.Elem)
	if done {
		return
	}
	if self.Left != nil {
		done = self.Left.doReverse(fn)
	}
	return
}

// DoMatch performs fn on all intervals stored in the tree that match q according to Overlap, with
// q.Overlap() used to guide tree traversal, so DoMatching() will out perform Do() with a called
// conditional function if the condition is based on sort order, but can not be reliably used if
// the condition is independent of sort order. A boolean is returned indicating whether the Do
// traversal was interrupted by an Operation returning true. If fn alters stored intervals' sort
// relationships, future tree operation behaviors are undefined.
func (self *Tree) DoMatching(fn Operation, q Overlapper) (t bool, err error) {
	if q.Min().Compare(q.Max()) > 0 {
		return false, ErrInvertedRange
	}
	if self.Root == nil {
		return
	}
	return self.Root.doMatch(fn, q), nil
}

func (self *Node) doMatch(fn Operation, q Overlapper) (done bool) {
	c := q.Overlap(self.Range)
	if c <= 0 && self.Left != nil {
		done = self.Left.doMatch(fn, q)
		if done {
			return
		}
	}
	if q.Overlap(self.Elem) == 0 {
		done = fn(self.Elem)
		if done {
			return
		}
	}
	if c >= 0 && self.Right != nil {
		done = self.Right.doMatch(fn, q)
	}
	return
}

// DoMatchReverse performs fn on all intervals stored in the tree that match q according to Overlap,
// with q.Overlap() used to guide tree traversal, so DoMatching() will out perform Do() with a called
// conditional function if the condition is based on sort order, but can not be reliably used if
// the condition is independent of sort order. A boolean is returned indicating whether the Do
// traversal was interrupted by an Operation returning true. If fn alters stored intervals' sort
// relationships, future tree operation behaviors are undefined.
func (self *Tree) DoMatchingReverse(fn Operation, q Overlapper) (t bool, err error) {
	if q.Min().Compare(q.Max()) > 0 {
		return false, ErrInvertedRange
	}
	if self.Root == nil {
		return
	}
	return self.Root.doMatchReverse(fn, q), nil
}

func (self *Node) doMatchReverse(fn Operation, q Overlapper) (done bool) {
	c := q.Overlap(self.Range)
	if c >= 0 && self.Right != nil {
		done = self.Right.doMatchReverse(fn, q)
		if done {
			return
		}
	}
	if q.Overlap(self.Elem) == 0 {
		done = fn(self.Elem)
		if done {
			return
		}
	}
	if c <= 0 && self.Left != nil {
		done = self.Left.doMatchReverse(fn, q)
	}
	return
}
