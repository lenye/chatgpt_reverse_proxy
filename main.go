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
	"syscall"
	"time"

	"github.com/lenye/chatgpt_reverse_proxy/internal/target"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/proxy"
	"github.com/lenye/chatgpt_reverse_proxy/pkg/version"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "print version string")
	)

	if envTarget, ok := os.LookupEnv("TARGET"); ok {
		if envTarget != "" {
			target.Url = envTarget
		}
	}

	flag.Parse()
	if *showVersion {
		fmt.Print(version.Print())
		return
	}

	log.Printf("reverse proxy target: %s", target.Url)

	oxyTarget, err := url.Parse(target.Url)
	if err != nil {
		log.Fatal(err)
	}

	oxy := proxy.BuildSingleHostProxy(oxyTarget)

	port := "9000"
	log.Printf("server on port %s", port)

	srv := &http.Server{Addr: ":" + port, Handler: oxy}

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
