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
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/lenye/chatgpt_reverse_proxy/internal/config"
)

type FakeApiKeyConfig struct {
	ApiKeys []string
}

type fakeApiKeyAuth struct {
	apiKeys map[string]bool
}

func FakeApiKey(cfg *FakeApiKeyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			fa := &fakeApiKeyAuth{
				apiKeys: getApiKeys(cfg.ApiKeys),
			}

			var (
				apiKeyHeader string
				apikey       string
			)

			switch config.APIType {
			case config.APITypeAzure:
				apiKeyHeader = config.AzureAPIKeyHeader
				apikey = r.Header.Get(apiKeyHeader)
				if apikey == "" {
					log.Printf("authentication failed, missing http header[%s]", apiKeyHeader)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
			case config.APITypeOpenAI, config.APITypeAzureAD:
				apiKeyHeader = config.APIKeyHeader
				apikey = r.Header.Get(apiKeyHeader)
				if !strings.HasPrefix(apikey, "Bearer ") {
					log.Printf("authentication failed, invalid http header[%s]", apiKeyHeader)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				hKeys := strings.Split(apikey, " ")
				if len(hKeys) != 2 {
					log.Printf("authentication failed, invalid http header[%s]", apiKeyHeader)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				apikey = hKeys[1]
				if apikey == "" {
					log.Printf("authentication failed, missing http header[%s]", apiKeyHeader)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
			}
			if apikey == "" {
				log.Println("authentication failed, missing http header auth")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if _, ok := fa.apiKeys[apikey]; !ok {
				log.Printf("authentication failed, invalid apikey: %s", apikey)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Removing authorization header
			r.Header.Del(apiKeyHeader)

			// Call the next middleware/handler in chain
			next.ServeHTTP(w, r)
		})
	}
}

func getApiKeys(apiKeys []string) map[string]bool {
	objMap := make(map[string]bool)
	for _, apikey := range apiKeys {
		objMap[apikey] = true
	}
	return objMap
}

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.ApiKey != "" {
			switch config.APIType {
			case config.APITypeAzure:
				r.Header.Set(config.AzureAPIKeyHeader, config.ApiKey)
			case config.APITypeOpenAI, config.APITypeAzureAD:
				r.Header.Set(config.APIKeyHeader, fmt.Sprintf("Bearer %s", config.ApiKey))
			}
		}
		next.ServeHTTP(w, r)
	})
}
