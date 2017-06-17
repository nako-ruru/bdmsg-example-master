// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package protodef defines default conventions for protocols.

	request
		charset : utf-8
		contenttype : application/json

	response
		charset : utf-8
		contenttype : application/json

	code
		1 : ok
		2 : run out

*/
package protodef

import (
	"io"
	"io/ioutil"

	. "common/datadef"
	. "common/errdef"
)

const (
	CodeResponseOK = 1
	BodyResponseOK = `{"code":1}`

	CodeResponseRunOut = 2
	InfoResponseRunOut = "run out"
	BodyResponseRunOut = `{"code":2,"info":"run out"}`
)

const (
	CheckStatusApi = "/checkstatus"
)

const (
	DefaultRequestContentType  = "application/json; charset=utf-8"
	DefaultResponseContentType = "application/json; charset=utf-8"
)

func ReadUnmarshal(r io.Reader, o Unmarshaler) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = o.Unmarshal(b)
	if err != nil {
		return err
	}

	return nil
}

func ReadUnmarshalValid(r io.Reader, o ValidUnmarshaler) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = o.Unmarshal(b)
	if err != nil {
		return err
	}

	if !o.Valid() {
		return ErrParameter
	}

	return nil
}

func MarshalWrite(w io.Writer, o Marshaler) error {
	b, err := o.Marshal()
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}
