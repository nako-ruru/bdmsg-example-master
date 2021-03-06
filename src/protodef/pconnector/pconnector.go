// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package pconnector defines the protocol of connector.

*/
package pconnector

import (
	"encoding/json"
	"sync"
	"compress/zlib"
	"bytes"
)

const (
	MsgTypeRegister 			= 0
	MsgTypeChat 				= 1
	MsgTypeEnterRoom 			= 4
	MsgTypeRefreshToken 		= 10
	MsgTypeHeartBeat			= 11
	MsgTypePush 				= 30001
)

type Register struct {
	UserId                   string
	Token                    string
	ClientToConnectorVersion int
}

func (p *Register) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Register) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type RefreshToken struct {
	Token string
}

func (p *RefreshToken) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type HeartBeat struct {
	ClientTime int64
}

func (p *HeartBeat) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type Chat struct {
	RoomId string
	Content string
	Nickname string
	Level int
	ClientTime int64
}

func (p *Chat) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type EnterRoom struct {
	RoomId string
	Nickname string
	Level int
}

func (p *EnterRoom) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type FromRouterMessage struct {
	MessageId string				`json:"messageId"`
	Time int64						`json:"time"`
	TimeText string					`json:"timeText"`

	ToUserId string 				`json:"toUserId"`
	ToRoomId string					`json:"toRoomId"`
	Params   map[string]string		`json:"params"`
	//Deprecated
	UserId string					`json:"userId"`
	//Deprecated
	RoomId string					`json:"roomId"`
	//Deprecated
	Content  string					`json:"content"`
}

func (p *FromRouterMessage) Unmarshal(b []byte) error {
	result := json.Unmarshal(b, p)
	if result != nil {
		if p.UserId != "" && p.ToUserId == "" {
			p.ToUserId = p.UserId
		}
		if p.RoomId != "" && p.ToRoomId == "" {
			p.ToRoomId = p.RoomId
		}
		v, ok := p.Params["content"]
		if p.Content != "" && (!ok || v == "") {
			p.ToUserId = v
		}
	}
	return result
}


type ToClientMessage struct {
	MessageId string				`json:"messageId"`
	ToUserId string 				`json:"toUserId"`
	ToRoomId string					`json:"toRoomId"`
	Params   map[string]string		`json:"params"`
	Time	 int64					`json:"time"`
	TimeText string					`json:"timeText"`
	//Deprecated
	UserId string					`json:"userId"`
	//Deprecated
	RoomId string					`json:"roomId"`
	//Deprecated
	Content  string					`json:"content"`
	Importance int					`json:"content"`

	serializedBytes []byte
	lock sync.RWMutex
}

func (p *ToClientMessage) Marshal() ([]byte, error) {
	p.lock.RLock()
	byteArray := p.serializedBytes
	p.lock.RUnlock()
	if byteArray != nil {
		return byteArray, nil
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	byteArray = p.serializedBytes
	var err error
	if byteArray == nil {
		var rawBytes []byte
		rawBytes, err = json.Marshal(p)
		if err == nil {
			compressed := p.doZlibCompress(rawBytes)
			p.serializedBytes = append([]byte{1}, compressed...)
			byteArray = p.serializedBytes
		}
	}
	return byteArray, err
}


func (s *ToClientMessage) doZlibCompress(src []byte) []byte {
	in := bytes.Buffer{}
	var w = zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}