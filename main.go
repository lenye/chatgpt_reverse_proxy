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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lenye/chatgpt_reverse_proxy/internal/config"
	"github.com/lenye/chatgpt_reverse_proxy/internal/env"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/alice"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/middleware/apikey"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/middleware/auth"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/proxy"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/version"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "print version string")
	)

	flag.Parse()
	if *showVersion {
		fmt.Print(version.Print())
		return
	}

	config.GetEnv()

	// Target
	oxyTarget, err := url.Parse(config.Target)
	if err != nil {
		log.Fatalf("invalid %s=%s, cause%s", env.Target, config.Target, err)
	}
	oxy := proxy.BuildSingleHostProxy(oxyTarget)

	// middleware
	chain := alice.New()
	switch config.AuthType {
	case config.AuthTypeBasic:
		cfg := auth.BasicConfig{
			Users: strings.Split(config.AuthBasicUsers, ","),
		}
		chain = chain.Append(auth.Basic(&cfg))
	case config.AuthTypeForward:
		cfg := auth.ForwardConfig{
			Address:            config.AuthForwardUrl,
			TrustForwardHeader: true,
		}
		chain = chain.Append(auth.Forward(&cfg))
	}
	chain = chain.Append(apikey.Handler)

	// http server
	srv := &http.Server{Addr: ":" + config.Port, Handler: chain.Then(oxy)}

	idleConnClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigint
		log.Printf("received signal: %v", s)

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("server shutdown failed: %v", err)
		}
		close(idleConnClosed)
	}()

	if err := srv.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("server closed")
		} else {
			log.Printf("server failed, err: %v", err)
		}
	}

	<-idleConnClosed

	log.Printf("%s exit, start: %s, uptime: %s",
		version.AppName, version.StartTime, time.Since(version.StartTime))
}
