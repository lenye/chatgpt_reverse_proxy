// Copyright 2023 The chatgpt_reverse_proxy Authors. All rights reserved.
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

package auth

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"
)

const (
	forwardedTypeName = "ForwardedAuthType"
	xForwardedURI     = "X-Forwarded-Uri"
	xForwardedMethod  = "X-Forwarded-Method"
	XForwardedProto   = "X-Forwarded-Proto"
	XForwardedFor     = "X-Forwarded-For"
	XForwardedHost    = "X-Forwarded-Host"
	XForwardedPort    = "X-Forwarded-Port"
	XForwardedServer  = "X-Forwarded-Server"
	XRealIP           = "X-Real-Ip"
)

const (
	Connection         = "Connection"
	KeepAlive          = "Keep-Alive"
	ProxyAuthenticate  = "Proxy-Authenticate"
	ProxyAuthorization = "Proxy-Authorization"
	Te                 = "Te" // canonicalized version of "TE"
	Trailers           = "Trailers"
	TransferEncoding   = "Transfer-Encoding"
	Upgrade            = "Upgrade"
	ContentLength      = "Content-Length"
)

const (
	connectionHeader = "Connection"
	upgradeHeader    = "Upgrade"
)

// hopHeaders Hop-by-hop headers to be removed in the authentication request.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
// Proxy-Authorization header is forwarded to the authentication server (see https://tools.ietf.org/html/rfc7235#section-4.4).
var hopHeaders = []string{
	Connection,
	KeepAlive,
	Te, // canonicalized version of "TE"
	Trailers,
	TransferEncoding,
	Upgrade,
}

type ForwardConfig struct {
	// Address defines the authentication server address.
	Address string
	// TrustForwardHeader defines whether to trust (ie: forward) all X-Forwarded-* headers.
	TrustForwardHeader bool
	// AuthResponseHeaders defines the list of headers to copy from the authentication server response and set on forwarded request, replacing any existing conflicting headers.
	AuthResponseHeaders []string
	// AuthResponseHeadersRegex defines the regex to match headers to copy from the authentication server response and set on forwarded request, after stripping all headers that match the regex.
	AuthResponseHeadersRegex string
	// AuthRequestHeaders defines the list of the headers to copy from the request to the authentication server.
	// If not set or empty then all request headers are passed.
	AuthRequestHeaders []string
}

type forwardAuth struct {
	address                  string
	authResponseHeaders      []string
	authResponseHeadersRegex *regexp.Regexp
	next                     http.Handler
	name                     string
	client                   http.Client
	trustForwardHeader       bool
	authRequestHeaders       []string
}

// Forward 转发认证
func Forward(config *ForwardConfig) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			fa := &forwardAuth{
				address:             config.Address,
				authResponseHeaders: config.AuthResponseHeaders,
				next:                next,
				trustForwardHeader:  config.TrustForwardHeader,
				authRequestHeaders:  config.AuthRequestHeaders,
			}

			// Ensure our request client does not follow redirects
			fa.client = http.Client{
				CheckRedirect: func(r *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 30 * time.Second,
			}

			if config.AuthResponseHeadersRegex != "" {
				re, err := regexp.Compile(config.AuthResponseHeadersRegex)
				if err != nil {
					log.Printf("error compiling regular expression %s: %s", config.AuthResponseHeadersRegex, err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				fa.authResponseHeadersRegex = re
			}

			// Remover removes hop-by-hop headers listed in the "Connection" header.
			// See RFC 7230, section 6.1.
			var reqUpType string
			if httpguts.HeaderValuesContainsToken(r.Header[connectionHeader], upgradeHeader) {
				reqUpType = r.Header.Get(upgradeHeader)
			}

			removeConnectionHeaders(r.Header)

			if reqUpType != "" {
				r.Header.Set(connectionHeader, upgradeHeader)
				r.Header.Set(upgradeHeader, reqUpType)
			} else {
				r.Header.Del(connectionHeader)
			}

			// 转发验证
			forwardReq, err := http.NewRequest(http.MethodGet, fa.address, nil)
			// tracing.LogRequest(tracing.GetSpan(req), forwardReq)
			if err != nil {
				log.Printf("error calling %s. cause %s", fa.address, err)

				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			writeHeader(r, forwardReq, fa.trustForwardHeader, fa.authRequestHeaders)

			forwardResponse, forwardErr := fa.client.Do(forwardReq)
			if forwardErr != nil {
				log.Printf("error calling %s. cause: %s", fa.address, forwardErr)

				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			body, readError := io.ReadAll(forwardResponse.Body)
			if readError != nil {
				log.Printf("error reading body %s. cause: %s", fa.address, readError)

				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer forwardResponse.Body.Close()

			// Pass the forward response's body and selected headers if it
			// didn't return a response within the range of [200, 300).
			if forwardResponse.StatusCode < http.StatusOK || forwardResponse.StatusCode >= http.StatusMultipleChoices {
				log.Printf("remote error %s. status code: %d", fa.address, forwardResponse.StatusCode)

				CopyHeaders(w.Header(), forwardResponse.Header)
				RemoveHeaders(w.Header(), hopHeaders...)

				// Grab the location header, if any.
				redirectURL, err := forwardResponse.Location()

				if err != nil {
					if !errors.Is(err, http.ErrNoLocation) {
						log.Printf("error reading response location header %s. cause: %s", fa.address, err)

						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				} else if redirectURL.String() != "" {
					// Set the location in our response if one was sent back.
					w.Header().Set("Location", redirectURL.String())
				}

				w.WriteHeader(forwardResponse.StatusCode)

				if _, err = w.Write(body); err != nil {
					log.Print(err)
				}
				return
			}

			for _, headerName := range fa.authResponseHeaders {
				headerKey := http.CanonicalHeaderKey(headerName)
				r.Header.Del(headerKey)
				if len(forwardResponse.Header[headerKey]) > 0 {
					r.Header[headerKey] = append([]string(nil), forwardResponse.Header[headerKey]...)
				}
			}

			if fa.authResponseHeadersRegex != nil {
				for headerKey := range r.Header {
					if fa.authResponseHeadersRegex.MatchString(headerKey) {
						r.Header.Del(headerKey)
					}
				}

				for headerKey, headerValues := range forwardResponse.Header {
					if fa.authResponseHeadersRegex.MatchString(headerKey) {
						r.Header[headerKey] = append([]string(nil), headerValues...)
					}
				}
			}

			r.RequestURI = r.URL.RequestURI()

			// Call the next middleware/handler in chain
			next.ServeHTTP(w, r)
		})
	}
}

func writeHeader(req, forwardReq *http.Request, trustForwardHeader bool, allowedHeaders []string) {
	CopyHeaders(forwardReq.Header, req.Header)
	RemoveHeaders(forwardReq.Header, hopHeaders...)

	forwardReq.Header = filterForwardRequestHeaders(forwardReq.Header, allowedHeaders)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if trustForwardHeader {
			if prior, ok := req.Header[XForwardedFor]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
		}
		forwardReq.Header.Set(XForwardedFor, clientIP)
	}

	xMethod := req.Header.Get(xForwardedMethod)
	switch {
	case xMethod != "" && trustForwardHeader:
		forwardReq.Header.Set(xForwardedMethod, xMethod)
	case req.Method != "":
		forwardReq.Header.Set(xForwardedMethod, req.Method)
	default:
		forwardReq.Header.Del(xForwardedMethod)
	}

	xfp := req.Header.Get(XForwardedProto)
	switch {
	case xfp != "" && trustForwardHeader:
		forwardReq.Header.Set(XForwardedProto, xfp)
	case req.TLS != nil:
		forwardReq.Header.Set(XForwardedProto, "https")
	default:
		forwardReq.Header.Set(XForwardedProto, "http")
	}

	if xfp := req.Header.Get(XForwardedPort); xfp != "" && trustForwardHeader {
		forwardReq.Header.Set(XForwardedPort, xfp)
	}

	xfh := req.Header.Get(XForwardedHost)
	switch {
	case xfh != "" && trustForwardHeader:
		forwardReq.Header.Set(XForwardedHost, xfh)
	case req.Host != "":
		forwardReq.Header.Set(XForwardedHost, req.Host)
	default:
		forwardReq.Header.Del(XForwardedHost)
	}

	xfURI := req.Header.Get(xForwardedURI)
	switch {
	case xfURI != "" && trustForwardHeader:
		forwardReq.Header.Set(xForwardedURI, xfURI)
	case req.URL.RequestURI() != "":
		forwardReq.Header.Set(xForwardedURI, req.URL.RequestURI())
	default:
		forwardReq.Header.Del(xForwardedURI)
	}
}

func filterForwardRequestHeaders(forwardRequestHeaders http.Header, allowedHeaders []string) http.Header {
	if len(allowedHeaders) == 0 {
		return forwardRequestHeaders
	}

	filteredHeaders := http.Header{}
	for _, headerName := range allowedHeaders {
		values := forwardRequestHeaders.Values(headerName)
		if len(values) > 0 {
			filteredHeaders[http.CanonicalHeaderKey(headerName)] = append([]string(nil), values...)
		}
	}

	return filteredHeaders
}
