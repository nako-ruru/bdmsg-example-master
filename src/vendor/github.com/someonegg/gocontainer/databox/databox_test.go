// Copyright 2016 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databox

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDataBoxReserve(t *testing.T) {
	at := assert.New(t)

	db := NewDataBox(16)

	// 0 invalid.
	at.False(db.Put(nil).valid())
	pos0, dst := db.Reserve(-1)
	at.False(pos0.valid())
	at.Nil(dst)

	// 1 first.
	pos1, dst := db.Reserve(4)
	at.Equal(DataPos{0, 0, 4}, pos1.innr)
	at.NotNil(dst)
	at.Equal(12, db.dir.Load().(*boxDir).remain)

	// 2 same cell.
	pos2, dst := db.Reserve(6)
	at.Equal(DataPos{0, 4, 6}, pos2.innr)
	at.NotNil(dst)
	at.Equal(6, db.dir.Load().(*boxDir).remain)

	// 3 remain 0.
	pos3, dst := db.Reserve(6)
	at.Equal(DataPos{0, 10, 6}, pos3.innr)
	at.NotNil(dst)
	at.Equal(0, db.dir.Load().(*boxDir).remain)

	// 4 new cell.
	pos4, dst := db.Reserve(8)
	at.Equal(DataPos{1, 0, 8}, pos4.innr)
	at.NotNil(dst)
	at.Equal(8, db.dir.Load().(*boxDir).remain)

	// 5 remain not enough.
	pos5, dst := db.Reserve(10)
	at.Equal(DataPos{2, 0, 10}, pos5.innr)
	at.NotNil(dst)
	at.Equal(6, db.dir.Load().(*boxDir).remain)

	// 6 bigger.
	pos6, dst := db.Reserve(18)
	at.Equal(DataPos{3, 0, 18}, pos6.innr)
	at.NotNil(dst)
	at.Equal(0, db.dir.Load().(*boxDir).remain)

	at.Equal(0, db.dir.Load().(*boxDir).base)

	db.Clean(pos4.innr)

	// 7 after clean.
	pos7, dst := db.Reserve(12)
	at.Equal(DataPos{4, 0, 12}, pos7.innr)
	at.NotNil(dst)
	at.Equal(4, db.dir.Load().(*boxDir).remain)

	at.Equal(1, db.dir.Load().(*boxDir).base)
}

func TestDataBoxGet(t *testing.T) {
	at := assert.New(t)

	db := NewDataBox(4)

	pos1 := db.Put(Data("a"))
	pos2 := db.Put(Data("bb"))
	pos3 := db.Put(Data("c"))
	pos4 := db.Put(Data("dd"))
	pos5 := db.Put(Data("eee"))
	pos6 := db.Put(Data("fffff"))

	// 0 invalid.
	at.Equal(Data(nil), db.Get(invalidBoxPos))

	// 1 first.
	at.Equal("a", string(db.Get(pos1)))

	// 2 same cell.
	at.Equal("bb", string(db.Get(pos2)))

	// 3 remain 0.
	at.Equal("c", string(db.Get(pos3)))

	// 4 new cell.
	at.Equal("dd", string(db.Get(pos4)))

	// 5 remain not enough.
	at.Equal("eee", string(db.Get(pos5)))

	// 6 bigger.
	at.Equal("fffff", string(db.Get(pos6)))

	db.Clean(pos4)

	pos7 := db.Put(Data("gggg"))

	// 7 after clean.
	at.Equal(Data(nil), db.Get(pos1))
	at.Equal(Data(nil), db.Get(pos2))
	at.Equal(Data(nil), db.Get(pos3))
	at.Equal("dd", string(db.Get(pos4)))
	at.Equal("eee", string(db.Get(pos5)))
	at.Equal("fffff", string(db.Get(pos6)))
	at.Equal("gggg", string(db.Get(pos7)))
}

func TestDataBoxClean(t *testing.T) {
	at := assert.New(t)

	db := NewDataBox(16)

	pos1 := db.Put(make(Data, 4))
	pos2 := db.Put(make(Data, 6))
	pos3 := db.Put(make(Data, 6))
	pos4 := db.Put(make(Data, 8))
	pos5 := db.Put(make(Data, 10))
	pos6 := db.Put(make(Data, 18))

	at.Equal(0, db.dir.Load().(*boxDir).base)

	// 1 no clean.
	at.Equal(0, db.Clean(pos1))
	at.Equal(0, db.Clean(pos2))
	at.Equal(0, db.Clean(pos3))
	at.Equal(0, db.dir.Load().(*boxDir).base)

	// 2 clean one.
	at.Equal(1, db.Clean(pos4))
	at.Equal(1, db.dir.Load().(*boxDir).base)
	at.Equal(1, db.Clean(pos5))
	at.Equal(2, db.dir.Load().(*boxDir).base)
	at.Equal(1, db.Clean(pos6))
	at.Equal(3, db.dir.Load().(*boxDir).base)

	// 3 already cleaned.
	at.Equal(0, db.Clean(pos4))
	at.Equal(3, db.dir.Load().(*boxDir).base)

	posX := DataPos{100, 0, 1}

	// 4 keep one .a.
	at.Equal(0, db.Clean(posX))
	at.Equal(3, db.dir.Load().(*boxDir).base)

	// 5 keep one .b.
	db.Put(make(Data, 10))
	db.Put(make(Data, 18))
	at.Equal(2, db.Clean(posX))
	at.Equal(5, db.dir.Load().(*boxDir).base)
}
