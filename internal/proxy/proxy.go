// Copyright 2023-2024 The chatgpt_reverse_proxy Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"

	"github.com/lenye/chatgpt_reverse_proxy/internal/config"
)

const DefaultFlushInterval = 100 * time.Millisecond

func BuildSingleHostProxy(target *url.URL) http.Handler {
	return &httputil.ReverseProxy{
		Director:      directorBuilder(target, false),
		FlushInterval: DefaultFlushInterval,
		BufferPool:    newBufferPool(),
		ErrorHandler:  errorHandler,
	}
}

func directorBuilder(target *url.URL, passHostHeader bool) func(req *http.Request) {
	return func(outReq *http.Request) {
		outReq.URL.Scheme = target.Scheme
		outReq.URL.Host = target.Host

		u := outReq.URL
		if outReq.RequestURI != "" {
			parsedURL, err := url.ParseRequestURI(outReq.RequestURI)
			if err == nil {
				u = parsedURL
			}
		}

		outReq.URL.Path = u.Path
		outReq.URL.RawPath = u.RawPath
		outReq.URL.RawQuery = strings.ReplaceAll(u.RawQuery, ";", "&")
		outReq.RequestURI = "" // Outgoing request should not have RequestURI

		outReq.Proto = "HTTP/1.1"
		outReq.ProtoMajor = 1
		outReq.ProtoMinor = 1

		// Do not pass client Host header unless optsetter PassHostHeader is set.
		if !passHostHeader {
			outReq.Host = outReq.URL.Host
		}

		cleanWebSocketHeaders(outReq)

		config.RemoveHop(outReq.Header)
	}
}

// cleanWebSocketHeaders Even if the websocket RFC says that headers should be case-insensitive,
// some servers need Sec-WebSocket-Key, Sec-WebSocket-Extensions, Sec-WebSocket-Accept,
// Sec-WebSocket-Protocol and Sec-WebSocket-Version to be case-sensitive.
// https://tools.ietf.org/html/rfc6455#page-20
func cleanWebSocketHeaders(req *http.Request) {
	if !isWebSocketUpgrade(req) {
		return
	}

	req.Header["Sec-WebSocket-Key"] = req.Header["Sec-Websocket-Key"]
	delete(req.Header, "Sec-Websocket-Key")

	req.Header["Sec-WebSocket-Extensions"] = req.Header["Sec-Websocket-Extensions"]
	delete(req.Header, "Sec-Websocket-Extensions")

	req.Header["Sec-WebSocket-Accept"] = req.Header["Sec-Websocket-Accept"]
	delete(req.Header, "Sec-Websocket-Accept")

	req.Header["Sec-WebSocket-Protocol"] = req.Header["Sec-Websocket-Protocol"]
	delete(req.Header, "Sec-Websocket-Protocol")

	req.Header["Sec-WebSocket-Version"] = req.Header["Sec-Websocket-Version"]
	delete(req.Header, "Sec-Websocket-Version")
}

func isWebSocketUpgrade(req *http.Request) bool {
	return httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") &&
		strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
}

func errorHandler(w http.ResponseWriter, req *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}

	statusCode := http.StatusInternalServerError

	if errors.Is(err, io.EOF) {
		statusCode = http.StatusBadGateway
	} else {
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				statusCode = http.StatusGatewayTimeout
			} else {
				statusCode = http.StatusBadGateway
			}
		}
	}

	w.WriteHeader(statusCode)
	if _, werr := w.Write([]byte(http.StatusText(statusCode))); werr != nil {
		slog.Error("errorHandler write status text failed", "error", werr)
	}
}
