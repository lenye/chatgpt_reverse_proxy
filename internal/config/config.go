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

package config

import (
	"log"
	"net/url"
	"os"
	"unicode"

	"github.com/lenye/chatgpt_reverse_proxy/internal/env"
)

var (
	Target = "https://api.openai.com"
	Port   = "9000"

	APIType = APITypeOpenAI
	ApiKey  = ""

	AuthType                   = ""
	AuthFakeApiKeys            = ""
	AuthForwardUrl             = ""
	AuthForwardRequestHeaders  = ""
	AuthForwardResponseHeaders = ""
)

const (
	APITypeOpenAI  = "open_ai"
	APITypeAzure   = "azure"
	APITypeAzureAD = "azure_ad"
)

const (
	APIKeyHeader      = "Authorization" // OpenAI or Azure AD authentication
	AzureAPIKeyHeader = "api-key"       // Azure authentication
)

const (
	AuthTypeFakeApiKey = "fake_apikey"
	AuthTypeForward    = "forward"
)

func isDigit(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func Read() {
	// Target
	if val, ok := os.LookupEnv(env.Target); ok {
		if val != "" {
			Target = val
		}
	}
	log.Printf("target: %s", Target)

	// Port
	if val, ok := os.LookupEnv(env.Port); ok {
		if val != "" {
			if !isDigit(val) {
				log.Fatalf("invalid %s: %s", env.Port, val)
			}
			Port = val
		}
	}
	log.Printf("serve on port %s", Port)

	// ApiKey
	if val, ok := os.LookupEnv(env.ApiKey); ok {
		if val != "" {
			ApiKey = val
		}
	}
	if ApiKey != "" {
		log.Printf("api key: %s", ApiKey)
	}

	// APIType
	if val, ok := os.LookupEnv(env.APIType); ok {
		switch val {
		case APITypeOpenAI, APITypeAzure, APITypeAzureAD:
			APIType = val
		}
	}
	log.Printf("api type: %s", APIType)

	// AuthType
	if val, ok := os.LookupEnv(env.AuthType); ok {
		switch val {
		case AuthTypeFakeApiKey:
			AuthType = val
			if ApiKey == "" {
				log.Fatalf("env %s missed", env.ApiKey)
			}
		}
	}
	if AuthType != "" {
		log.Printf("auth type: %s", AuthType)
	}

	switch AuthType {
	case AuthTypeFakeApiKey:
		// AuthFakeApiKeys
		if val, ok := os.LookupEnv(env.AuthFakeApiKeys); ok {
			if val != "" {
				AuthFakeApiKeys = val
			}
		}
		if AuthFakeApiKeys == "" {
			log.Fatalf("env %s missed", env.AuthFakeApiKeys)
		}
		log.Printf("fake api keys: %s", AuthFakeApiKeys)
	case AuthTypeForward:
		// AuthForwardUrl
		if val, ok := os.LookupEnv(env.AuthForwardUrl); ok {
			if val != "" {
				AuthForwardUrl = val
			}
		}
		if AuthForwardUrl == "" {
			log.Fatalf("env %s missed", env.AuthForwardUrl)
		}
		if _, err := url.Parse(AuthForwardUrl); err != nil {
			log.Fatalf("invalid %s=%s, cause %s", env.AuthForwardUrl, AuthForwardUrl, err)
		}
		log.Printf("auth forward url: %s", AuthForwardUrl)

		// AuthForwardRequestHeaders
		if val, ok := os.LookupEnv(env.AuthForwardRequestHeaders); ok {
			if val != "" {
				AuthForwardRequestHeaders = val
			}
		}

		// AuthForwardResponseHeaders
		if val, ok := os.LookupEnv(env.AuthForwardResponseHeaders); ok {
			if val != "" {
				AuthForwardResponseHeaders = val
			}
		}
	}
}
