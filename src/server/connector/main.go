// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/someonegg/golog"
	"github.com/someonegg/goutil/gologf"
	"github.com/someonegg/goutil/pidf"
	"github.com/someonegg/gox/netx"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"common/service/debugsvc"
	. "server/connector/internal/config"
	"server/connector/internal/manager"
	"server/connector/internal/service/connectsvc"
)

var log = golog.SubLoggerWithFields(golog.RootLogger, "module", "main")

func main() {
	rand.Seed(time.Now().Unix())
	golog.RootLogger.AddPredef("app", "connector")

	err := ParseConfig()
	if err != nil {
		log.Error("main$ParseConfig", "error", err)
		return
	}

	err = gologf.SetOutput(Config.Logfile)
	if err != nil {
		log.Error("main$SetOutput", "error", err)
		return
	}

	mSet := manager.NewManagerSet(&Config.Manager)

	clientM := connectsvc.NewClientManager(mSet)

	connectS, err := connectsvc.Start(
		&Config.ServiceS.Connect.BDMsgSvcConfT, clientM)
	if err != nil {
		log.Error("main$connectsvc.Start", "error", err)
		return
	}

	var debugS *netx.HTTPService
	if Config.ServiceS.Debug.Check() {
		debugS, err = debugsvc.Start(&Config.ServiceS.Debug)
		if err != nil {
			log.Error("main$debugsvc.Start", "error", err)
		}
	}

	pidF := pidf.New(Config.Pidfile)
	defer pidF.Close()

	log.Info("connector started", "pid", pidF.Pid)

	// Handle SIGINT and SIGTERM.
	qC := make(chan os.Signal, 1)
	signal.Notify(qC, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-qC:
		log.Info(s.String())
	case <-connectS.StopD():
		log.Fatal("connectsvc stopped", "error", connectS.Err())
	}

	if debugS != nil {
		debugS.Stop()
		debugS.WaitRequests()
	}

	connectS.Stop()
	clientM.CloseAll()
	connectS.WaitClients()

	mSet.Close()

	time.Sleep(1 * time.Second)
}
