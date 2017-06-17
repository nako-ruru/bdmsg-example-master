// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skiplist_test

import (
	"fmt"
	"github.com/someonegg/gocontainer/skiplist"
	//"math/rand"
	//"time"
)

type elem struct {
	skiplist.Node
	v int
}

func elemCompare(l, r skiplist.Scorable) int {
	vl := int(0)
	switch v := l.(type) {
	case int:
		vl = v
	case *elem:
		vl = v.v
	default:
		panic(l)
	}
	vr := int(0)
	switch v := r.(type) {
	case int:
		vr = v
	case *elem:
		vr = v.v
	default:
		panic(r)
	}
	return vl - vr
}

func Example() {
	//rand.Seed(time.Now().Unix())

	// An empty skiplist and put some numbers in it.
	l := skiplist.NewList(elemCompare)

	// add and print rank
	e1 := &elem{v: 4}
	l.Add(e1)
	fmt.Println(l.Rank(e1))

	e2 := &elem{v: 1}
	l.Add(e2)
	fmt.Println(l.Rank(e2))

	e3 := &elem{v: 3}
	l.Add(e3)
	fmt.Println(l.Rank(e3))

	e4 := &elem{v: 2}
	l.Add(e4)
	fmt.Println(l.Rank(e4))

	e5 := &elem{v: 4}
	l.Add(e5)
	fmt.Println(l.Rank(e5))

	e6 := &elem{v: 6}
	l.Add(e6)
	fmt.Println(l.Rank(e6))

	fmt.Println()

	// print list after add.
	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Println(e.(*elem).v)
	}

	fmt.Println()

	l.Remove(l.Find(6))
	l.Remove(l.Get(4))

	// print list after remove.
	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Println(e.(*elem).v)
	}

	// Output:
	// 0
	// 0
	// 1
	// 1
	// 3
	// 5
	//
	// 1
	// 2
	// 3
	// 4
	// 4
	// 6
	//
	// 1
	// 2
	// 3
	// 4
}
