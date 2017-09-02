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
	MsgTypeSupport = 2
	MsgTypeSendGift = 3
	MsgTypeEnterRoom = 4
	MsgTypeShare = 5
	MsgTypeLevelUp = 6

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


type PushMsg struct {
	UserId  string 			`json:"userId"`
	RoomId string			`json:"roomId"`
	ModuleId  string		`json:"moduleId"`
	Content string			`json:"content"`
}

func (p *PushMsg) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *PushMsg) Unmarshal(b []byte) error {
	return json.Unmarshal(b, p)
}
