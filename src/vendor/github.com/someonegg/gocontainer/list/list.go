// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package list implements a doubly linked list.
// The list's method is similar to standard "container/list".
// The main difference is:
//   list management structure needs to be embedded into your element.
package list

// Element interface is used to traverse the list.
type Element interface {
	// Next returns the next list element or nil.
	Next() Element
	// Prev returns the previous list element or nil.
	Prev() Element

	// Raw method, for list package.
	getNode() *Node
}

// Node is the ONLY implementation of the Element interface, it
// needs to be embedded into your element structure.
type Node struct {
	next Element
	prev Element
	list *List
}

func (d *Node) Next() Element {
	if p := d.next; d.list != nil && p != &d.list.root {
		return p
	}
	return nil
}

func (d *Node) Prev() Element {
	if p := d.prev; d.list != nil && p != &d.list.root {
		return p
	}
	return nil
}

func (d *Node) getNode() *Node {
	return d
}

type List struct {
	root Node
	len  int
}

func (l *List) Init() *List {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func New() *List { return new(List).Init() }

func (l *List) Len() int { return l.len }

func (l *List) Front() Element {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

func (l *List) Back() Element {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

func (l *List) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

func (l *List) insert(e, at Element) Element {
	eN := e.getNode()
	atN := at.getNode()
	n := atN.next
	nN := n.getNode()
	atN.next = e
	eN.prev = at
	eN.next = n
	nN.prev = e
	eN.list = l
	l.len++
	return e
}

func (l *List) remove(e Element) Element {
	eN := e.getNode()
	n := eN.next
	nN := n.getNode()
	p := eN.prev
	pN := p.getNode()
	pN.next = n
	nN.prev = p
	eN.next = nil
	eN.prev = nil
	eN.list = nil
	l.len--
	return e
}

func (l *List) Remove(e Element) Element {
	if e.getNode().list == l {
		l.remove(e)
	}
	return e
}

func (l *List) PushFront(e Element) Element {
	l.lazyInit()
	return l.insert(e, &l.root)
}

func (l *List) PushBack(e Element) Element {
	l.lazyInit()
	return l.insert(e, l.root.prev)
}

func (l *List) InsertBefore(e Element, mark Element) Element {
	if mark.getNode().list != l {
		return nil
	}
	return l.insert(e, mark.getNode().prev)
}

func (l *List) InsertAfter(e Element, mark Element) Element {
	if mark.getNode().list != l {
		return nil
	}
	return l.insert(e, mark)
}

func (l *List) MoveToFront(e Element) {
	if e.getNode().list != l || l.root.next == e {
		return
	}
	l.insert(l.remove(e), &l.root)
}

func (l *List) MoveToBack(e Element) {
	if e.getNode().list != l || l.root.prev == e {
		return
	}
	l.insert(l.remove(e), l.root.prev)
}

func (l *List) MoveBefore(e, mark Element) {
	if e.getNode().list != l || e == mark || mark.getNode().list != l {
		return
	}
	l.insert(l.remove(e), mark.getNode().prev)
}

func (l *List) MoveAfter(e, mark Element) {
	if e.getNode().list != l || e == mark || mark.getNode().list != l {
		return
	}
	l.insert(l.remove(e), mark)
}

// MergeBackList merge an other list at the back of list l.
// The other list will be reset.
func (l *List) MergeBackList(other *List) {
	l.lazyInit()

	if l == other {
		return
	}

	for {
		e := other.Front()
		if e == nil {
			break
		}
		other.remove(e)
		l.insert(e, l.root.prev)
	}
}

// MergeFrontList merge an other list at the front of list l.
// The other list will be reset.
func (l *List) MergeFrontList(other *List) {
	l.lazyInit()

	if l == other {
		return
	}

	for {
		e := other.Back()
		if e == nil {
			break
		}
		other.remove(e)
		l.insert(e, &l.root)
	}
}
