// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errdef

import "errors"

var (
	ErrUnknown      = errors.New("unknown")
	ErrCancel       = errors.New("cancel")
	ErrPanic        = errors.New("panic")
	ErrUnexpected   = errors.New("unexpected")
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExist = errors.New("already exist")

	ErrParameter = errors.New("wrong parameter")
	ErrKey       = errors.New("wrong key")
	ErrAddress   = errors.New("address wrong")
)
