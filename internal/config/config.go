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

	AuthType       = ""
	AuthBasicUsers = ""
	AuthForwardUrl = ""
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
	AuthTypeBasic   = "basic"
	AuthTypeForward = "forward"
)

func isDigit(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func GetEnv() {
	// Target
	if envTarget, ok := os.LookupEnv(env.Target); ok {
		if envTarget != "" {
			Target = envTarget
		}
	}
	log.Printf("target: %s", Target)

	// Port
	if envPort, ok := os.LookupEnv(env.Port); ok {
		if envPort != "" {
			if !isDigit(envPort) {
				log.Fatalf("invalid %s: %s", env.Port, envPort)
			}
			Port = envPort
		}
	}
	log.Printf("serve on port %s", Port)

	// ApiKey
	if envApiKey, ok := os.LookupEnv(env.ApiKey); ok {
		if envApiKey != "" {
			ApiKey = envApiKey
		}
	}
	if ApiKey != "" {
		log.Printf("api key: %s", ApiKey)
	}

	// APIType
	if envAPIType, ok := os.LookupEnv(env.APIType); ok {
		switch envAPIType {
		case APITypeOpenAI, APITypeAzure, APITypeAzureAD:
			APIType = envAPIType
		}
	}
	log.Printf("api type: %s", APIType)

	// AuthType
	if envAuthType, ok := os.LookupEnv(env.AuthType); ok {
		switch envAuthType {
		case AuthTypeBasic, AuthTypeForward:
			AuthType = envAuthType
			if ApiKey == "" {
				log.Fatalf("env %s missed", env.ApiKey)
			}
		}
	}
	if AuthType != "" {
		log.Printf("auth type: %s", AuthType)
	}

	switch AuthType {
	case AuthTypeBasic:
		// AuthBasicUsers
		if envAuthBasicUsers, ok := os.LookupEnv(env.AuthBasicUsers); ok {
			if envAuthBasicUsers != "" {
				AuthBasicUsers = envAuthBasicUsers
			}
		}
		if AuthBasicUsers == "" {
			log.Fatalf("env %s missed", env.AuthBasicUsers)
		}
		log.Printf("basic auth users: %s", AuthBasicUsers)
	case AuthTypeForward:
		// AuthForwardUrl
		if envAuthForwardUrl, ok := os.LookupEnv(env.AuthForwardUrl); ok {
			if envAuthForwardUrl != "" {
				AuthForwardUrl = envAuthForwardUrl
			}
		}
		if AuthForwardUrl == "" {
			log.Fatalf("env %s missed", env.AuthForwardUrl)
		}
		if _, err := url.Parse(AuthForwardUrl); err != nil {
			log.Fatalf("invalid %s=%s, cause %s", env.AuthForwardUrl, AuthForwardUrl, err)
		}
		log.Printf("auth forward url: %s", AuthForwardUrl)
	}
}
