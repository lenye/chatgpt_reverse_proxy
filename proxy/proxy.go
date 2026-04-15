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

	"github.com/lenye/chatgpt_reverse_proxy/config"
)

const DefaultFlushInterval = 100 * time.Millisecond

func BuildSingleHostProxy(target *url.URL, passHostHeader bool, preservePath bool) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite:       rewriteBuilder(target, passHostHeader, preservePath),
		Transport:     httpTransport,
		FlushInterval: DefaultFlushInterval,
		BufferPool:    newBufferPool(),
		ErrorHandler:  errorHandler,
	}
}

func rewriteBuilder(target *url.URL, passHostHeader bool, preservePath bool) func(*httputil.ProxyRequest) {
	return func(pr *httputil.ProxyRequest) {
		copyForwardedHeader(pr.Out.Header, pr.In.Header)
		if clientIP, _, err := net.SplitHostPort(pr.In.RemoteAddr); err == nil {
			// If we aren't the first proxy retain prior
			// X-Forwarded-For information as a comma+space
			// separated list and fold multiple headers into one.
			prior, ok := pr.Out.Header["X-Forwarded-For"]
			omit := ok && prior == nil // Issue 38079: nil now means don't populate the header
			if len(prior) > 0 {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
			if !omit {
				pr.Out.Header.Set("X-Forwarded-For", clientIP)
			}
		}

		pr.Out.URL.Scheme = target.Scheme
		pr.Out.URL.Host = target.Host

		u := pr.Out.URL
		if pr.Out.RequestURI != "" {
			parsedURL, err := url.ParseRequestURI(pr.Out.RequestURI)
			if err == nil {
				u = parsedURL
			}
		}

		pr.Out.URL.Path = u.Path
		pr.Out.URL.RawPath = u.RawPath

		if preservePath {
			pr.Out.URL.Path, pr.Out.URL.RawPath = JoinURLPath(target, u)
		}

		// If a plugin/middleware adds semicolons in query params, they should be urlEncoded.
		pr.Out.URL.RawQuery = strings.ReplaceAll(u.RawQuery, ";", "&")
		pr.Out.RequestURI = "" // Outgoing request should not have RequestURI

		pr.Out.Proto = "HTTP/1.1"
		pr.Out.ProtoMajor = 1
		pr.Out.ProtoMinor = 1

		// Do not pass client Host header unless option PassHostHeader is set.
		if !passHostHeader {
			pr.Out.Host = pr.Out.URL.Host
		}

		if isWebSocketUpgrade(pr.Out) {
			cleanWebSocketHeaders(pr.Out)
		}

		config.RemoveHop(pr.Out.Header)
	}
}

// copyForwardedHeader copies header that are removed by the reverseProxy when a rewriteRequest is used.
func copyForwardedHeader(dst, src http.Header) {
	if val, ok := src["X-Forwarded-For"]; ok {
		dst["X-Forwarded-For"] = val
	}
	if val, ok := src["Forwarded"]; ok {
		dst["Forwarded"] = val
	}
	if val, ok := src["X-Forwarded-Host"]; ok {
		dst["X-Forwarded-Host"] = val
	}
	if val, ok := src["X-Forwarded-Proto"]; ok {
		dst["X-Forwarded-Proto"] = val
	}
}

// JoinURLPath computes the joined path and raw path of the given URLs.
// From https://github.com/golang/go/blob/b521ebb55a9b26c8824b219376c7f91f7cda6ec2/src/net/http/httputil/reverseproxy.go#L221
func JoinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}

	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
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
		if netErr, ok := errors.AsType[net.Error](err); ok {
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
