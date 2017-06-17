// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package datadef defines some common types.
package datadef

type Valider interface {
	Valid() bool
}

type Marshaler interface {
	Marshal() ([]byte, error)
}

type ValidMarshaler interface {
	Valider
	Marshaler
}

type Unmarshaler interface {
	Unmarshal([]byte) error
}

type ValidUnmarshaler interface {
	Valider
	Unmarshaler
}
