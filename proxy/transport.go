package proxy

import (
	"net"
	"net/http"
	"time"
)

var httpTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,

	IdleConnTimeout: 30 * time.Second,

	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,

	TLSHandshakeTimeout:    5 * time.Second,
	ResponseHeaderTimeout:  10 * time.Second,
	ExpectContinueTimeout:  1 * time.Second,
	WriteBufferSize:        32 * 1024,
	ReadBufferSize:         32 * 1024,
	MaxResponseHeaderBytes: 32 * 1024,
	ForceAttemptHTTP2:      true,
}
