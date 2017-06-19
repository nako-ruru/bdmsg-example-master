// Copyright 2016 SHAHE. All rights reserved.

package httpcli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/someonegg/golog"
	"github.com/someonegg/goutil/netutil"
	"golang.org/x/net/context"
	"net/http"
	"time"

	. "common/config"
	. "common/datadef"
	. "protodef"
)

var (
	ErrRemoteServer = errors.New("remote server internal error")
)

type CallOption struct {
	f func(*callOptions)
}

type callOptions struct {
	url   string
	addr  string
	https bool
}

// use the url directly
func WithURL(url string) CallOption {
	return CallOption{func(o *callOptions) {
		o.url = url
	}}
}

// use addr as host
func WithAddr(addr string) CallOption {
	return CallOption{func(o *callOptions) {
		o.addr = addr
	}}
}

// use https
func WithHttps() CallOption {
	return CallOption{func(o *callOptions) {
		o.https = true
	}}
}

type BaseProxy struct {
	defAddr string
	client  *netutil.HttpClient
	logger  golog.Logger
}

func NewBaseProxy(conf *UpstreamConfT, logger golog.Logger) *BaseProxy {
	return &BaseProxy{
		defAddr: conf.Addr,
		client:  netutil.NewHttpClient(conf.MaxConcurrent, time.Duration(conf.Timeout)),
		logger:  logger,
	}
}

func (p *BaseProxy) Close() error {
	return p.client.Close()
}

func (p *BaseProxy) Call(ctx context.Context, api string,
	param Marshaler, result Unmarshaler, options ...CallOption) error {

	if ctx == nil {
		ctx = context.Background()
	}

	o := callOptions{}
	for _, option := range options {
		option.f(&o)
	}

	var url string
	if len(o.url) > 0 {
		url = o.url
	} else {
		addr := p.defAddr
		if len(o.addr) > 0 {
			addr = o.addr
		}
		if o.https {
			url = fmt.Sprintf("https://%v%v", addr, api)
		} else {
			url = fmt.Sprintf("http://%v%v", addr, api)
		}
	}

	var (
		resp *http.Response
		err  error
	)

	if param != nil {
		var b bytes.Buffer
		MarshalWrite(&b, param)
		resp, err = p.client.Post(ctx, url, DefaultRequestContentType, &b)
	} else {
		resp, err = p.client.Get(ctx, url)
	}
	if err != nil {
		p.logger.Error("Call$Post_Get", "error", err, "url", url)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		p.logger.Error("Call$StatusCode", "error", resp.StatusCode, "url", url)
		return ErrRemoteServer
	}

	if result != nil {
		err = ReadUnmarshal(resp.Body, result)
		if err != nil {
			p.logger.Error("Call$ReadUnmarshal", "error", err, "url", url)
			return err
		}
	}

	return nil
}
