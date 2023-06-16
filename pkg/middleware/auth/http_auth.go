// From the project https://github.com/abbot/go-http-auth

package auth

import (
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
)

const contentType = "Content-Type"

// SecretProvider is used by authenticators. Takes user name and realm
// as an argument, returns secret required for authentication (HA1 for
// digest authentication, properly encrypted password for basic).
//
// Returning an empty string means failing the authentication.
type SecretProvider func(user, realm string) string

// Headers contains header and error codes used by authenticator.
type Headers struct {
	Authenticate      string // WWW-Authenticate
	Authorization     string // Authorization
	AuthInfo          string // Authentication-Info
	UnauthCode        int    // 401
	UnauthContentType string // text/plain
	UnauthResponse    string // Unauthorized.
}

// V returns NormalHeaders when h is nil, or h otherwise. Allows to
// use uninitialized *Headers values in structs.
func (h *Headers) V() *Headers {
	if h == nil {
		return NormalHeaders
	}
	return h
}

var (
	// NormalHeaders are the regular Headers used by an HTTP Server for
	// request authentication.
	NormalHeaders = &Headers{
		Authenticate:      "WWW-Authenticate",
		Authorization:     "Authorization",
		AuthInfo:          "Authentication-Info",
		UnauthCode:        http.StatusUnauthorized,
		UnauthContentType: "text/plain",
		UnauthResponse:    fmt.Sprintf("%d %s\n", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)),
	}
)

// BasicAuth is an authenticator implementation for 'Basic' HTTP
// Authentication scheme (RFC 7617).
type BasicAuth struct {
	Realm   string
	Secrets SecretProvider
	// Headers used by authenticator. Set to ProxyHeaders to use with
	// proxy server. When nil, NormalHeaders are used.
	Headers *Headers
}

// CopyHeaders copies http headers from source to destination, it
// does not override, but adds multiple headers.
func CopyHeaders(dst http.Header, src http.Header) {
	for k, vv := range src {
		dst[k] = append(dst[k], vv...)
	}
}

// RemoveHeaders removes the header with the given names from the headers map.
func RemoveHeaders(headers http.Header, names ...string) {
	for _, h := range names {
		headers.Del(h)
	}
}

func removeConnectionHeaders(h http.Header) {
	for _, f := range h[connectionHeader] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}
