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

package env

const Prefix = "OXY_"

const (
	Port           = Prefix + "PORT"
	Target         = Prefix + "TARGET"
	APIType        = Prefix + "API_TYPE"
	ApiKey         = Prefix + "API_KEY"
	AuthType       = Prefix + "AUTH_TYPE"
	AuthBasicUsers = Prefix + "AUTH_BASIC_USERS"
	AuthForwardUrl = Prefix + "AUTH_FORWARD_URL"
)
