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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/lenye/chatgpt_reverse_proxy/internal/env"
)

var Target = "https://api.openai.com"

func Read() error {
	// Target
	if val, ok := os.LookupEnv(env.Target); ok {
		if val != "" {
			Target = val
		}
	}

	// WebPort
	if val, ok := os.LookupEnv(env.WebPort); ok {
		if val != "" {
			uInt, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				return fmt.Errorf("invalid %s=%s, cause: %w", env.WebPort, val, err)
			}
			if uInt > 65535 {
				return fmt.Errorf("invalid %s=%s, cause: max=65535", env.WebPort, val)
			}
			WebPort = val
		}
	}

	return nil
}

func RemoveHop(header http.Header) {
	if HopPrefix != "" {
		hop := make([]string, 0)
		for k := range header {
			kk := strings.ToUpper(k)
			if strings.HasPrefix(kk, HopPrefix) {
				hop = append(hop, k)
			}
		}
		for _, h := range hop {
			header.Del(h)
		}
	}
}
