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
	"bytes"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultRealm        = "reverse_proxy"
	authorizationHeader = "Authorization"
)

type BasicConfig struct {
	// Users is an array of authorized users.
	// Each user must be declared using the name:hashed-password format.
	// Tip: Use htpasswd to generate the passwords.
	Users []string
}

type basicAuth struct {
	users map[string]string
	auth  *BasicAuth
}

func (b *basicAuth) secretBasic(user, realm string) string {
	if secret, ok := b.users[user]; ok {
		return secret
	}

	return ""
}

// RequireAuth is an http.HandlerFunc for BasicAuth which initiates
// the authentication process (or requires reauthentication).
func (a *BasicAuth) RequireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentType, a.Headers.V().UnauthContentType)
	w.Header().Set(a.Headers.V().Authenticate, `Basic realm="`+a.Realm+`"`)
	w.WriteHeader(a.Headers.V().UnauthCode)
	_, _ = w.Write([]byte(a.Headers.V().UnauthResponse))
}

// Basic Basic认证
func Basic(config *BasicConfig) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			users, err := getUsers(config.Users, basicUserParser)
			if err != nil {
				log.Printf("error getUsers, cause: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			realm := defaultRealm
			ba := &basicAuth{
				users: users,
			}
			ba.auth = &BasicAuth{Realm: realm, Secrets: ba.secretBasic}

			user, password, ok := r.BasicAuth()
			if ok {
				secret := ba.auth.Secrets(user, ba.auth.Realm)
				if secret == "" || !CheckSecret(password, secret) {
					ok = false
				}
			}

			if !ok {
				log.Println("authentication failed")
				ba.auth.RequireAuth(w, r)
				return
			}

			// Authentication succeeded
			//r.URL.User = url.User(user)

			// Removing authorization header
			r.Header.Del(authorizationHeader)

			// Call the next middleware/handler in chain
			next.ServeHTTP(w, r)
		})
	}
}

// UserParser Parses a string and return a userName/userHash. An error if the format of the string is incorrect.
type UserParser func(user string) (string, string, error)

func getUsers(users []string, parser UserParser) (map[string]string, error) {
	userMap := make(map[string]string)
	for _, user := range users {
		userName, userHash, err := parser(user)
		if err != nil {
			return nil, err
		}
		userMap[userName] = userHash
	}

	return userMap, nil
}

func basicUserParser(user string) (string, string, error) {
	split := strings.Split(user, ":")
	if len(split) != 2 {
		return "", "", fmt.Errorf("error parsing BasicUser: %v", user)
	}
	return split[0], split[1], nil
}

type compareFunc func(hashedPassword, password []byte) error

var (
	errMismatchedHashAndPassword = errors.New("mismatched hash and password")

	compareFuncs = []struct {
		prefix  string
		compare compareFunc
	}{
		{"", compareMD5HashAndPassword}, // default compareFunc
		{"{SHA}", compareShaHashAndPassword},
		// Bcrypt is complicated. According to crypt(3) from
		// crypt_blowfish version 1.3 (fetched from
		// http://www.openwall.com/crypt/crypt_blowfish-1.3.tar.gz), there
		// are three different has prefixes: "$2a$", used by versions up
		// to 1.0.4, and "$2x$" and "$2y$", used in all later
		// versions. "$2a$" has a known bug, "$2x$" was added as a
		// migration path for systems with "$2a$" prefix and still has a
		// bug, and only "$2y$" should be used by modern systems. The bug
		// has something to do with handling of 8-bit characters. Since
		// both "$2a$" and "$2x$" are deprecated, we are handling them the
		// same way as "$2y$", which will yield correct results for 7-bit
		// character passwords, but is wrong for 8-bit character
		// passwords. You have to upgrade to "$2y$" if you want sant 8-bit
		// character password support with bcrypt. To add to the mess,
		// OpenBSD 5.5. introduced "$2b$" prefix, which behaves exactly
		// like "$2y$" according to the same source.
		{"$2a$", bcrypt.CompareHashAndPassword},
		{"$2b$", bcrypt.CompareHashAndPassword},
		{"$2x$", bcrypt.CompareHashAndPassword},
		{"$2y$", bcrypt.CompareHashAndPassword},
	}
)

// CheckSecret returns true if the password matches the encrypted
// secret.
func CheckSecret(password, secret string) bool {
	compare := compareFuncs[0].compare
	for _, cmp := range compareFuncs[1:] {
		if strings.HasPrefix(secret, cmp.prefix) {
			compare = cmp.compare
			break
		}
	}
	return compare([]byte(secret), []byte(password)) == nil
}

func compareMD5HashAndPassword(hashedPassword, password []byte) error {
	parts := bytes.SplitN(hashedPassword, []byte("$"), 4)
	if len(parts) != 4 {
		return errMismatchedHashAndPassword
	}
	magic := []byte("$" + string(parts[1]) + "$")
	salt := parts[2]
	if subtle.ConstantTimeCompare(hashedPassword, MD5Crypt(password, salt, magic)) != 1 {
		return errMismatchedHashAndPassword
	}
	return nil
}

func compareShaHashAndPassword(hashedPassword, password []byte) error {
	d := sha1.New()
	d.Write(password)
	if subtle.ConstantTimeCompare(hashedPassword[5:], []byte(base64.StdEncoding.EncodeToString(d.Sum(nil)))) != 1 {
		return errMismatchedHashAndPassword
	}
	return nil
}
