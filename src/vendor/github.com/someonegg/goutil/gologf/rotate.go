// Copyright 2016 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

package gologf

import (
	"os"
	"os/signal"
	"syscall"
)

func init() {
	go logSig()
}

func logSig() {
	defer func() { recover() }()

	// SIGUSR1 to reload log.
	rC := make(chan os.Signal, 1)
	signal.Notify(rC, syscall.SIGUSR1)

	for {
		select {
		case <-rC:
			locker.Lock()
			path := logS
			locker.Unlock()
			if len(path) > 0 {
				SetOutput(path)
			}
		}
	}
}
