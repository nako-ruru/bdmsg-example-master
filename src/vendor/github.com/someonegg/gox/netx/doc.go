// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package netx provides net wrappers over the original package,
// "Context" and "concurrency control" is supported.
package netx

import (
	"net"
	"time"
)

// TcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections.
type TcpKeepAliveListener struct {
	*net.TCPListener
}

func (l TcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := l.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
