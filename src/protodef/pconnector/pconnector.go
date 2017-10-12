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
	MsgTypeRegister = 0
	MsgTypeChat 		= 1
	MsgTypeEnterRoom = 4
	MsgTypePush = 30000
)

type Register struct {
	UserId string
	Pass   string
}

func (p *Register) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Register) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type Chat struct {
	RoomId string
	Content string
	Nickname string
	Level int
	ClientTime uint64
}

func (p *Chat) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Chat) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type Support struct {
	RoomId string
	Nickname string
	Level int
}

func (p *Support) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Support) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type SendGift struct {
	RoomId string
	GiftId string
	Nickname string
	Level int
}

func (p *SendGift) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *SendGift) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type EnterRoom struct {
	RoomId string
	Nickname string
	Level int
}

func (p *EnterRoom) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *EnterRoom) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}

type Share struct {
	RoomId string
	Nickname string
	Level int
}

func (p *Share) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Share) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}


type LevelUp struct {
	RoomId string
	Nickname string
	Level int
}

func (p *LevelUp) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *LevelUp) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}


type FromRouterMessage struct {
	MessageId string				`json:messageId`
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
	MessageId string				`json:messageId`
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
}

func (p *ToClientMessage) Marshal() ([]byte, error) {
	return json.Marshal(p)
}
