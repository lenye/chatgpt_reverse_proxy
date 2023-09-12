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
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lenye/chatgpt_reverse_proxy/internal/config"
	"github.com/lenye/chatgpt_reverse_proxy/internal/env"
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
	start := time.Now()

	config.Read()

	// Target
	oxyTarget, err := url.Parse(config.Target)
	if err != nil {
		slog.Error(fmt.Sprintf("invalid %s=%s", env.Target, config.Target), "error", err)
		os.Exit(1)
		return
	}
	oxy := proxy.BuildSingleHostProxy(oxyTarget)

	srv := &http.Server{Addr: ":" + config.Port, Handler: oxy}

	idleConnClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigint
		slog.Debug(fmt.Sprintf("received signal: %v", s))

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			slog.Error("server shutdown failed", "error", err)
		}
		close(idleConnClosed)
	}()

	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			slog.Info("server closed")
		} else {
			slog.Error("server failed", "error", err)
		}
	}

	<-idleConnClosed

	slog.Info(version.AppName+" exit",
		"start", start, "uptime", time.Since(start),
	)
}
