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
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lenye/chatgpt_reverse_proxy/internal/config"
	"github.com/lenye/chatgpt_reverse_proxy/internal/env"
	"github.com/lenye/chatgpt_reverse_proxy/internal/proxy"
	"github.com/lenye/chatgpt_reverse_proxy/internal/version"
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

	if err := config.Read(); err != nil {
		slog.Error("config.Read failed", "error", err)
		os.Exit(1)
		return
	}

	// Target
	oxyTarget, err := url.Parse(config.Target)
	if err != nil {
		slog.Error(fmt.Sprintf("invalid %s=%s", env.Target, config.Target), "error", err)
		os.Exit(1)
		return
	}

	slog.Info("Configuration", slog.Group("config", "target", config.Target, "port", config.WebPort))

	var lc net.ListenConfig
	// 主动启用mptcp
	lc.SetMultipathTCP(true)

	isMultipathTCP := lc.MultipathTCP()
	slog.Debug("net.ListenConfig", "MultipathTCP", isMultipathTCP)

	ln, err := lc.Listen(context.Background(), "tcp", ":"+config.WebPort)
	if err != nil {
		slog.Error("server listen failed", "error", err)
		os.Exit(1)
		return
	}
	slog.Info("server listening on " + ln.Addr().String())

	oxy := proxy.BuildSingleHostProxy(oxyTarget)
	srv := &http.Server{Addr: ":" + config.WebPort, Handler: oxy}

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
	}()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "error", err)
	}
	slog.Info("server stopped")

	slog.Info(version.AppName+" exit",
		"start", start, "uptime", time.Since(start),
	)
}
