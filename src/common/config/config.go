// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"
)

var (
	ErrConfigContent  = errors.New("config content invalid")
	ErrDurationFormat = errors.New("duration format invalid")
	ErrTCPAddrFormat  = errors.New("tcpaddr format invalid")
)

type Duration time.Duration

func (d Duration) Check() bool {
	return d > 0
}

func (d Duration) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `"%v"`, time.Duration(d).String())
	return buf.Bytes(), nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return ErrDurationFormat
	}
	_d, err := time.ParseDuration(string(b[1 : len(b)-1]))
	if err != nil {
		return err
	}
	*d = Duration(_d)
	return nil
}

type TCPAddr net.TCPAddr

func (d *TCPAddr) Check() bool {
	return len(d.IP) > 0 && d.Port != 0
}

func (d *TCPAddr) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `"%v"`, (*net.TCPAddr)(d).String())
	return buf.Bytes(), nil
}

func (d *TCPAddr) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return ErrTCPAddrFormat
	}
	_d, err := net.ResolveTCPAddr("tcp", string(b[1:len(b)-1]))
	if err != nil {
		return err
	}
	*d = *(*TCPAddr)(_d)
	return nil
}

type ServiceConfT struct {
	ListenAddr    TCPAddr  `json:"listenaddr"`
	MaxConcurrent int      `json:"maxconcurrent"`
	HesitateTime  Duration `json:"hesitatetime"`
}

func (c *ServiceConfT) Check() bool {
	return c.ListenAddr.Check() && c.MaxConcurrent >= 0 && c.HesitateTime >= 0
}

type UpstreamConfT struct {
	Addr          string   `json:"address"`
	MaxConcurrent int      `json:"maxconcurrent"`
	Timeout       Duration `json:"timeout"`
}

func (c *UpstreamConfT) Check() bool {
	return len(c.Addr) > 0 && c.MaxConcurrent >= 0 && c.Timeout >= 0
}

type MysqlConfT struct {
	Addr          string `json:"address"`
	MaxConcurrent int    `json:"maxconcurrent"`
	User          string `json:"user"`
	Pass          string `json:"password"`
	DBName        string `json:"database"`
}

func (c *MysqlConfT) Check() bool {
	return len(c.Addr) > 0 && c.MaxConcurrent >= 0 && len(c.User) > 0 && len(c.Pass) > 0 && len(c.DBName) > 0
}

type RedisConfT struct {
	Addr          string `json:"address"`
	MaxConcurrent int    `json:"maxconcurrent"`
	Pass          string `json:"password"`
}

func (c *RedisConfT) Check() bool {
	return len(c.Addr) > 0 && c.MaxConcurrent >= 0
}

type BDMsgSvcConfT struct {
	ListenAddr  TCPAddr  `json:"listenaddr"`
	InqueueN    int      `json:"inqueuesize"`
	OutqueueN   int      `json:"outqueuesize"`
	HandshakeTO Duration `json:"handshaketimeout"`
}

func (c *BDMsgSvcConfT) Check() bool {
	return c.ListenAddr.Check() && c.InqueueN > 0 && c.OutqueueN > 0 && c.HandshakeTO > 0
}

type BDMsgCliConfT struct {
	Addr      TCPAddr `json:"address"`
	InqueueN  int     `json:"inqueuesize"`
	OutqueueN int     `json:"outqueuesize"`
}

func (c *BDMsgCliConfT) Check() bool {
	return c.Addr.Check() && c.InqueueN > 0 && c.OutqueueN > 0
}
