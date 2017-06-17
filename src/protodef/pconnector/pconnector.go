// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package pconnector defines the protocol of connector.

*/
package pconnector

import (
	"encoding/json"
)

const (
	MsgTypeConnect      = 1
	MsgTypeConnectReply = 2

	MsgTypeClientHello = 3

	MsgTypeServerHello = 4
)

type ConnectRequst struct {
	ID   string
	Pass string
}

func (p *ConnectRequst) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ConnectRequst) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type ConnectReply struct {
	Code  int
	Token string
}

func (p *ConnectReply) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ConnectReply) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type ClientHello struct {
	Message string
}

func (p *ClientHello) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ClientHello) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type ServerHello struct {
	Message string
}

func (p *ServerHello) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ServerHello) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}
