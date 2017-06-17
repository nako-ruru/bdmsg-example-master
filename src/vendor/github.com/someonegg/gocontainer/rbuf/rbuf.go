// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package rbuf implements simple ring buffer.
package rbuf

import (
	"errors"
	"io"
)

var (
	ErrBufferFull = errors.New("buffer is full")
)

// Fixed size ring buffer.
type FixedRingBuf struct {
	// capacity
	N int

	buf  []byte
	beg  int
	size int
}

// NewFixedRingBuf creates and initializes a new FixedRingBuf with the capacity.
// In most cases, new(FixedRingBuf) (or just declaring a FixedRingBuf variable) is
// sufficient to initialize a FixedRingBuf.
func NewFixedRingBuf(capacity int) *FixedRingBuf {
	t := &FixedRingBuf{N: capacity}
	t.Init()
	return t
}

func (b *FixedRingBuf) Init() {
	b.buf = make([]byte, b.N, b.N)
	b.beg = 0
	b.size = 0
}

func (b *FixedRingBuf) lazyInit() {
	if b.buf == nil {
		b.Init()
	}
}

func (b *FixedRingBuf) Reset() {
	b.beg = 0
	b.size = 0
}

func (b *FixedRingBuf) Len() int {
	return b.size
}

func (b *FixedRingBuf) Read(p []byte) (int, error) {
	return b.read(p, false)
}

func (b *FixedRingBuf) Peek(p []byte) (int, error) {
	return b.read(p, true)
}

func (b *FixedRingBuf) read(p []byte, peek bool) (n int, err error) {
	b.lazyInit()

	lp := len(p)

	if lp == 0 {
		return
	}
	if b.size == 0 {
		return 0, io.EOF
	}

	rbeg := b.beg
	rend := rbeg + b.size
	if rend <= b.N {
		n += copy(p, b.buf[rbeg:rend])
	} else {
		n += copy(p, b.buf[rbeg:b.N])
		if n < lp {
			n += copy(p[n:], b.buf[0:(rend%b.N)])
		}
	}

	if !peek {
		b.Skip(n)
	}

	return
}

func (b *FixedRingBuf) Skip(n int) {
	if n > b.size {
		n = b.size
	}
	if n <= 0 {
		return
	}
	b.size -= n
	b.beg = (b.beg + n) % b.N
}

// Return (len(p),nil) or (0,ErrBufferFull).
func (b *FixedRingBuf) Write(p []byte) (n int, err error) {
	b.lazyInit()

	lp := len(p)

	if lp == 0 {
		return
	}

	can := b.N - b.size
	if lp > can {
		return 0, ErrBufferFull
	}

	wbeg := (b.beg + b.size) % b.N
	wend := wbeg + can
	if wend <= b.N {
		n += copy(b.buf[wbeg:wend], p)
	} else {
		n += copy(b.buf[wbeg:b.N], p)
		if n < lp {
			n += copy(b.buf[0:(wend%b.N)], p[n:])
		}
	}

	b.size += n

	return
}

const DefaultGrowthUnit = 256

// Variable length ring buffer.
type RingBuf struct {
	FixedRingBuf

	// If zero, use DefaultGrowthUnit.
	GrowthUnit int
}

// NewRingBuf creates and initializes a new RingBuf with the capacity.
// In most cases, new(RingBuf) (or just declaring a RingBuf variable) is
// sufficient to initialize a RingBuf.
func NewRingBuf(capacity, growthUnit int) *RingBuf {
	t := &RingBuf{}
	t.N = capacity
	t.GrowthUnit = growthUnit
	t.Init()
	return t
}

// Return (len(p),nil).
func (b *RingBuf) Write(p []byte) (n int, err error) {
	if b.GrowthUnit == 0 {
		b.GrowthUnit = DefaultGrowthUnit
	}

	n, err = b.FixedRingBuf.Write(p)
	if err != ErrBufferFull {
		return
	}

	l := b.size + len(p)
	l = ((l-1)/b.GrowthUnit + 1) * b.GrowthUnit
	nb := make([]byte, l, l)
	b.FixedRingBuf.Peek(nb)

	b.N = l
	b.buf = nb
	b.beg = 0

	return b.FixedRingBuf.Write(p)
}
