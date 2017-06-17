// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue_test

import (
	"fmt"
	"github.com/someonegg/gocontainer/list"
	"github.com/someonegg/gocontainer/queue"
)

type elem struct {
	list.Node
	v int
}

func Example() {
	// An empty queue and put some numbers in it.
	var q queue.EventQueue
	q.Init(nil)
	q.PushFront(&elem{v: 1})
	q.PushFront(&elem{v: 2})
	q.PushFront(&elem{v: 3})
	q.PushFront(&elem{v: 4})

	select {
	case <-q.Event():
	}

	for e := q.PopFront(); e != nil; e = q.PopFront() {
		fmt.Println(e.(*elem).v)
	}

	// Output:
	// 4
	// 3
	// 2
	// 1
}
