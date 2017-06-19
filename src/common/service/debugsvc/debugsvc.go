// Copyright 2016 SHAHE. All rights reserved.

package debugsvc

import (
	"github.com/someonegg/golog"
	"github.com/someonegg/gox/netx"
	"net"
	"net/http"
	_ "net/http/pprof"

	. "common/config"
	. "common/errdef"
)

var log = golog.SubLoggerWithFields(golog.RootLogger, "module", "debugsvc")

type service struct {
	*netx.HTTPService
}

func newService(l *net.TCPListener) *service {
	s := &service{}
	s.HTTPService = netx.NewHTTPService(l, http.DefaultServeMux, 0)
	return s
}

func Start(conf *ServiceConfT) (*netx.HTTPService, error) {
	l, err := net.ListenTCP("tcp", (*net.TCPAddr)(&conf.ListenAddr))
	if err != nil {
		log.Error("Start$net.ListenTCP", "error", err)
		return nil, ErrAddress
	}

	s := newService(l)
	s.Start()

	return s.HTTPService, nil
}
