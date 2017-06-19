// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netx

import (
	"github.com/someonegg/gox/syncx"
	"golang.org/x/net/context"
	"io"
	"net"
	. "net/http"
	"net/url"
	"time"
)

// HTTPClient is a contexted http client.
type HTTPClient struct {
	ts     *Transport
	hc     *Client
	concur syncx.Semaphore
}

// if maxConcurrent == 0, no limit on concurrency.
func NewHTTPClient(maxConcurrent int, timeout time.Duration) *HTTPClient {
	mi := maxConcurrent / 5
	if mi <= 0 {
		mi = DefaultMaxIdleConnsPerHost
	}
	ts := &Transport{
		Proxy: ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConnsPerHost: mi,
	}
	hc := &Client{
		Transport: ts,
		Timeout:   timeout,
	}

	c := &HTTPClient{}
	c.ts = ts
	c.hc = hc
	if maxConcurrent > 0 {
		c.concur = syncx.NewSemaphore(maxConcurrent)
	}
	return c
}

func (c *HTTPClient) acquireConn(ctx context.Context) error {
	if c.concur == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	// Acquire
	case c.concur <- struct{}{}:
		return nil
	}
}

func (c *HTTPClient) releaseConn() {
	if c.concur == nil {
		return
	}

	<-c.concur
}

func (c *HTTPClient) Do(ctx context.Context,
	req *Request) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Do(req)
}

func (c *HTTPClient) Get(ctx context.Context,
	url string) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Get(url)
}

func (c *HTTPClient) Head(ctx context.Context,
	url string) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Head(url)
}

func (c *HTTPClient) Post(ctx context.Context,
	url string, bodyType string, body io.Reader) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.Post(url, bodyType, body)
}

func (c *HTTPClient) PostForm(ctx context.Context,
	url string, data url.Values) (resp *Response, err error) {

	err = c.acquireConn(ctx)
	if err != nil {
		return
	}
	defer c.releaseConn()

	return c.hc.PostForm(url, data)
}

func (c *HTTPClient) Close() error {
	c.ts.CloseIdleConnections()
	return nil
}
