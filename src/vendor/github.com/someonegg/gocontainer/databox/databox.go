// Copyright 2016 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package databox defines the DataBox type, which can be used to store
// data to reduce the number of references and memory fragmentation.
package databox

import (
	"sync/atomic"
)

// DataBox internal:
//   Box has an infinite number of cells, each cell has a index (0 1 2 ...).
//   Data will be copied into a cell.
//   When the current cell can not put the new Data, enable the next one.
//   Cell size has a default value, if less than the new Data, use Data size.
// DataBox support for "single-write + multi-read" without locker.
type DataBox struct {
	cellSize int

	// dir *boxDir
	dir atomic.Value
}

// NewDataBox creates and initializes a new DataBox with the cellSize.
func NewDataBox(cellSize int) *DataBox {
	t := &DataBox{cellSize: cellSize}
	t.dir.Store(&boxDir{})
	return t
}

// Data is a variable-sized buffer of bytes.
type Data []byte

// DataPos is the data position, can be used to retrieve data (Get).
type DataPos struct {
	cidx int
	cpos int
	size int
}

func (p DataPos) valid() bool {
	return p.cidx >= 0 && p.cpos >= 0 && p.size > 0
}

var invalidBoxPos = DataPos{-1, -1, 0}

// RsvdPos is the reserved position, will be converted to DataPos when Commit.
type RsvdPos struct {
	innr DataPos
}

func (p RsvdPos) valid() bool {
	return p.innr.valid()
}

// Put will copy the Data to DataBox.
func (b *DataBox) Put(d Data) DataPos {
	if len(d) == 0 {
		return invalidBoxPos
	}

	p, dst := b.reserve(len(d))

	copy(dst, d)

	return p
}

// Reserve returns an n-sized Data, which can be used to write.
// When writing is complete, you need to Commit.
func (b *DataBox) Reserve(n int) (RsvdPos, Data) {
	if n <= 0 {
		return RsvdPos{invalidBoxPos}, nil
	}

	p, dst := b.reserve(n)

	return RsvdPos{p}, dst
}

func (b *DataBox) Commit(p RsvdPos) DataPos {
	return p.innr
}

// Get can retrieve data. If the cell is cleaned, returns nil.
func (b *DataBox) Get(p DataPos) Data {
	if !p.valid() {
		return nil
	}

	bdir := b.dir.Load().(*boxDir)

	if !(p.cidx >= bdir.base && p.cidx-bdir.base < len(bdir.cells)) {
		return nil
	}

	c := bdir.cells[p.cidx-bdir.base]

	if !(p.cpos < len(c) && p.size <= len(c)-p.cpos) {
		return nil
	}

	return Data(c[p.cpos : p.cpos+p.size])
}

// Clean will remove the cell before the "to" position (not strict),
// returns the number of cells cleaned.
func (b *DataBox) Clean(to DataPos) int {
	bdir := b.dir.Load().(*boxDir)

	if to.cidx <= bdir.base {
		return 0
	}
	if len(bdir.cells) <= 1 {
		// keep one at least.
		return 0
	}

	skip := to.cidx - bdir.base
	if skip >= len(bdir.cells) {
		skip = len(bdir.cells) - 1
	}

	n := &boxDir{}
	n.base = bdir.base + skip
	n.cells = make([]cell, len(bdir.cells)-skip)
	copy(n.cells, bdir.cells[skip:])
	n.remain = bdir.remain

	b.dir.Store(n)

	return skip
}

type cell []byte

type boxDir struct {
	base   int    // Copy-on-write.
	cells  []cell // Copy-on-write.
	remain int
}

func (b *DataBox) reserve(size int) (DataPos, []byte) {
	bdir := b.dir.Load().(*boxDir)

	r := bdir.remain
	if r >= size {
		bdir.remain = r - size

		cidx := len(bdir.cells) - 1
		c := bdir.cells[cidx] // current.
		cidx += bdir.base
		cpos := len(c) - r
		return DataPos{cidx, cpos, size}, c[cpos : cpos+size]
	}

	// alloc new cell.

	r = size
	if r < b.cellSize {
		r = b.cellSize
	}

	n := &boxDir{}
	n.base = bdir.base
	n.cells = make([]cell, len(bdir.cells)+1)
	copy(n.cells, bdir.cells)
	c := make(cell, r)
	n.cells[len(n.cells)-1] = c
	n.remain = r - size

	cidx := n.base + len(n.cells) - 1

	b.dir.Store(n)

	return DataPos{cidx, 0, size}, c[0:size]
}
