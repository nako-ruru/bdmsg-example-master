// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package skiplist implements a skip list. Compared with the classical
// version, there are two changes:
//   this implementation allows for repeated elements.
//   there is a back pointer, so it's a doubly linked list.
// List will be sorted by score:
//   in ascending order.
//   with rank(0-based), also in ascending order.
//   "what is the score, how to compare" is defined by the user.
package skiplist

import (
	"fmt"
	"math/rand"
)

// The fixed probability.
const PROPABILITY float32 = 0.25

// The default level limit of skip list.
var DefaultMaxLevel int = 32

// Use math/rand pkg's default Source.
type pseudoRandSource struct{}

func (r pseudoRandSource) Int63() int64 {
	return rand.Int63()
}

func (r pseudoRandSource) Seed(seed int64) {
	rand.Seed(seed)
}

// The default rand source.
var DefaultRandSource rand.Source = pseudoRandSource{}

// Scorable object can be passed to CompareFunc.
type Scorable interface{}

// CompareFunc can compare two scorable objects, returns
//   <0 if l <  r
//    0 if l == r
//   >0 if l >  r
type CompareFunc func(l, r Scorable) int

// Element interface is used to traverse the list.
//
// List element must be scorable.
type Element interface {
	// Next returns the next list element or nil.
	Next() Element
	// Prev returns the previous list element or nil.
	Prev() Element

	// Raw method, for skiplist package.
	getNode() *Node
}

type level struct {
	next Element
	prev Element
	span int
}

// Node is the ONLY implementation of the Element interface, it
// needs to be embedded into your element structure.
type Node struct {
	lev  []level
	list *List
}

func (d *Node) Next() Element {
	if p := d.lev[0].next; d.list != nil && p.getNode() != d.list.root {
		return p
	}
	return nil
}

func (d *Node) Prev() Element {
	if p := d.lev[0].prev; d.list != nil && p.getNode() != d.list.root {
		return p
	}
	return nil
}

func (d *Node) getNode() *Node {
	return d
}

type List struct {
	mlev int
	rnds rand.Source
	comp CompareFunc
	root *Node
	len  int
}

// Creates a new skip list, with DefaultMaxLevel\DefaultRandSource\compare.
func NewList(compare CompareFunc) *List {
	return NewListEx(DefaultMaxLevel, DefaultRandSource, compare)
}

// Creates a new skip list, with maxLevel\randSource\compare.
func NewListEx(maxLevel int, randSource rand.Source, compare CompareFunc) *List {
	if maxLevel < 1 {
		panic("maxLevel < 1")
	}
	if randSource == nil {
		panic("randSource is nil")
	}
	if compare == nil {
		panic("compare is nil")
	}

	l := &List{
		mlev: maxLevel,
		rnds: randSource,
		comp: compare,
		root: &Node{
			lev:  make([]level, maxLevel),
			list: nil,
		},
	}

	for i := 0; i < l.mlev; i++ {
		l.root.lev[i].next = l.root
		l.root.lev[i].prev = l.root
		l.root.lev[i].span = 0
	}

	return l
}

// Len returns the number of elements of list l. The complexity is O(1).
func (l *List) Len() int { return l.len }

// Front returns the first element of list l or nil.
func (l *List) Front() Element {
	if l.len == 0 {
		return nil
	}
	return l.root.lev[0].next
}

// Back returns the last element of list l or nil.
func (l *List) Back() Element {
	if l.len == 0 {
		return nil
	}
	return l.root.lev[0].prev
}

// Get the element at rank, return nil if rank is invalid.
//   0 <= valid rank < list.Len()
func (l *List) Get(rank int) Element {
	if rank < 0 || rank >= l.len {
		return nil
	}

	e, found := l.searchToRank(rank, nil)
	if !found || e.getNode() == l.root {
		panic("impossible")
	}

	return e
}

// Find the first element equal to score, return nil if not found.
// If there are multiple elements equal to score, you can use the
// "Element" to traverse them.
func (l *List) Find(score Scorable) Element {
	if score == nil {
		return nil
	}

	e, found := l.searchToScore(score, nil)
	if found && e.getNode() == l.root {
		panic("impossible")
	}

	if !found {
		return nil
	}
	return e
}

// Calculate current rank of the element, return -1 if not in the list.
func (l *List) Rank(e Element) int {
	eN := e.getNode()
	if eN.list != l {
		return -1
	}

	path := l.searchPathOf(e)

	span := 0
	for _, v := range path.levSpan {
		span += v
	}

	return span - 1
}

// Add an element to the list.
func (l *List) Add(e Element) {
	l.add(e)
}

func (l *List) add(e Element) {
	path := l.newSearchPath()

	ee, found := l.searchToScore(e, path)
	if found && ee.getNode() == l.root {
		panic("impossible")
	}

	randON := true

	// repeated element
	if found {
		if len(ee.getNode().lev) == 1 {
			// only 1 level, insert before.
			path.prev[0] = path.prevNode[0].lev[0].prev
			path.prevNode[0] = path.prev[0].getNode()
			path.levSpan[0] -= 1
		} else {
			// more than 1 level, force 1 level on newer.
			randON = false
		}
	}

	nlev := 1
	if randON {
		nlev = l.randLevel()
	}

	//fmt.Println(nlev, randON)

	eN := e.getNode()
	eN.lev = make([]level, nlev)

	revspan := 0
	for i := 0; i < nlev; i++ {
		p := path.prev[i]
		pN := path.prevNode[i]
		n := pN.lev[i].next
		nN := n.getNode()
		pN.lev[i].next = e
		eN.lev[i].prev = p
		eN.lev[i].next = n
		nN.lev[i].prev = e

		eN.lev[i].span = pN.lev[i].span - revspan
		pN.lev[i].span = revspan + 1
		revspan += path.levSpan[i]
	}

	for i := nlev; i < l.mlev; i++ {
		path.prevNode[i].lev[i].span += 1
	}

	eN.list = l
	l.len++
}

func (l *List) randLevel() int {
	const RANDMAX int64 = 65536
	const RANDTHRESHOLD int64 = int64(float32(RANDMAX) * PROPABILITY)
	nlev := 1
	for l.rnds.Int63()%RANDMAX < RANDTHRESHOLD && nlev <= l.mlev {
		nlev++
	}
	return nlev
}

// Remove an element from the list.
func (l *List) Remove(e Element) {
	eN := e.getNode()
	if eN.list != l {
		return
	}
	l.remove(e)
}

func (l *List) remove(e Element) {
	path := l.searchPathOf(e)

	eN := e.getNode()
	for i := 0; i < len(eN.lev); i++ {
		n := eN.lev[i].next
		nN := n.getNode()
		p := eN.lev[i].prev
		pN := p.getNode()
		pN.lev[i].next = n
		nN.lev[i].prev = p
		pN.lev[i].span += eN.lev[i].span - 1
	}

	for i := len(eN.lev); i < l.mlev; i++ {
		path.prevNode[i].lev[i].span -= 1
	}

	eN.lev = nil
	eN.list = nil
	l.len--
}

// searchPath represents search path of skip list.
type searchPath struct {
	prev     []Element
	prevNode []*Node // optimization
	levSpan  []int
}

func (l *List) newSearchPath() *searchPath {
	return &searchPath{
		prev:     make([]Element, l.mlev),
		prevNode: make([]*Node, l.mlev),
		levSpan:  make([]int, l.mlev),
	}
}

func (l *List) searchPathOf(e Element) *searchPath {
	path := l.newSearchPath()

	eN := e.getNode()
	path.prev[0] = e
	path.prevNode[0] = eN
	path.levSpan[0] = 0

	ilev := 0
	for {
		idle, levSpan := true, 0

		for i := ilev + 1; i < len(eN.lev); i++ {
			idle = false
			path.prev[i] = e
			path.prevNode[i] = eN
			path.levSpan[i] = 0
			ilev++
		}

		for eN != l.root && (ilev+1) >= len(eN.lev) {
			idle = false
			e = eN.lev[ilev].prev
			eN = e.getNode()
			levSpan += eN.lev[ilev].span
		}

		if idle {
			break
		}

		path.levSpan[ilev] = levSpan
	}

	return path
}

// return
//   <0 goto down
//    0 found
//   >0 goto next
type poscompFunc func(ilev int, p, n Element, pN, nN *Node) int

func (l *List) searchToPos(poscomp poscompFunc, path *searchPath) (Element, bool) {
	found := false

	p := Element(l.root)
	pN := p.getNode()

	for i := l.mlev - 1; i >= 0; i-- {
		levSpan := 0

		equal := false
		n := pN.lev[i].next
		nN := n.getNode()
		for nN != l.root {
			ret := poscomp(i, p, n, pN, nN)
			if ret < 0 {
				break
			}
			levSpan += pN.lev[i].span
			p = n
			pN = nN
			n = nN.lev[i].next
			nN = n.getNode()
			if ret == 0 {
				equal = true
				break
			}
		}

		if path != nil {
			path.prev[i] = p
			path.prevNode[i] = pN
			path.levSpan[i] = levSpan
		}

		if equal {
			found = true
			for i--; path != nil && i >= 0; i-- {
				path.prev[i] = p
				path.prevNode[i] = pN
				path.levSpan[i] = 0
			}
			break
		}
	}

	return p, found
}

// searchToXXX will find the element that is closest to XXX.
// If the "path" is not nil, it will be filled.

func (l *List) searchToScore(score Scorable, path *searchPath) (Element, bool) {
	poscomp := func(ilev int, p, n Element, pN, nN *Node) int {
		return l.comp(score, n)
	}

	return l.searchToPos(poscomp, path)
}

func (l *List) searchToRank(rank int, path *searchPath) (Element, bool) {
	span := rank + 1
	poscomp := func(ilev int, p, n Element, pN, nN *Node) int {
		ret := span - pN.lev[ilev].span
		if ret >= 0 {
			span = ret
		}
		return ret
	}
	return l.searchToPos(poscomp, path)
}

func (l *List) dump() {
	fmt.Println("TotalLevel:", l.mlev, " ", "Length:", l.len)
	fmt.Println()
	for i := l.mlev - 1; i >= 0; i-- {
		fmt.Println("Level:", i)
		e := Element(l.root)
		eN := e.getNode()
		fmt.Print("  ")
		for {
			fmt.Println("\t", eN.lev[i].span)

			e = eN.lev[i].next
			eN = e.getNode()
			if eN == l.root {
				break
			}
			fmt.Print(e, " ")
		}
		fmt.Println()
	}
}
