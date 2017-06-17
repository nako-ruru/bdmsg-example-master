// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list_test

import (
	"fmt"
	"github.com/someonegg/gocontainer/list"
)

type elem struct {
	list.Node
	v int
}

func Example() {
	// An empty list and put some numbers in it.
	var l list.List
	e4 := l.PushBack(&elem{v: 4})
	e1 := l.PushFront(&elem{v: 1})
	l.InsertBefore(&elem{v: 3}, e4)
	l.InsertAfter(&elem{v: 2}, e1)

	// Iterate through list and print its contents.
	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Println(e.(*elem).v)
	}

	// Output:
	// 1
	// 2
	// 3
	// 4
}
