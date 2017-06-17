// Copyright 2016 SHAHE. All rights reserved.

package debugsvc

import (
	"github.com/someonegg/golog"
	"github.com/someonegg/goutility/netutil"
	"net"
	"net/http"
	_ "net/http/pprof"

	. "common/config"
	. "common/errdef"
)

var log = golog.SubLoggerWithFields(golog.RootLogger, "module", "debugsvc")

type service struct {
	*netutil.HttpService
}

func newService(l *net.TCPListener) *service {
	s := &service{}
	s.HttpService = netutil.NewHttpService(l, http.DefaultServeMux, 0)
	return s
}

func Start(conf *ServiceConfT) (*netutil.HttpService, error) {
	l, err := net.ListenTCP("tcp", (*net.TCPAddr)(&conf.ListenAddr))
	if err != nil {
		log.Error("Start$net.ListenTCP", "error", err)
		return nil, ErrAddress
	}

	s := newService(l)
	s.Start()

	return s.HttpService, nil
}
