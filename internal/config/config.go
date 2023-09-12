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
	"log/slog"
	"os"
	"strconv"

	"github.com/lenye/chatgpt_reverse_proxy/internal/env"
)

var (
	Target = "https://api.openai.com"
	Port   = "9000"
)

func Read() {
	// Target
	if val, ok := os.LookupEnv(env.Target); ok {
		if val != "" {
			Target = val
		}
	}
	slog.Info(fmt.Sprintf("target: %s", Target))

	// Port
	if val, ok := os.LookupEnv(env.Port); ok {
		if val != "" {
			uInt, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				slog.Error(fmt.Sprintf("invalid %s=%s", env.Port, val), "error", err)
				os.Exit(1)
				return
			}
			if uInt > 65535 {
				slog.Error(fmt.Sprintf("invalid %s=%s, max=65535", env.Port, val))
				os.Exit(1)
				return
			}
			Port = val
		}
	}
	slog.Info(fmt.Sprintf("serve on port %s", Port))
}
