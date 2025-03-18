// Copyright 2025 The Authors (see AUTHORS file)
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

// Package main is the main entrypoint to the application.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abcxyz/abc-updater/pkg/metrics"
	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/tagrep/internal/metricswrap"
	"github.com/abcxyz/tagrep/internal/version"
	"github.com/abcxyz/tagrep/pkg/commands/issue"
	"github.com/abcxyz/tagrep/pkg/commands/request"
)

const (
	defaultLogLevel  = "warn"
	defaultLogFormat = "json"
	defaultLogDebug  = "false"
)

// rootCmd defines the starting command structure.
var rootCmd = func() cli.Command {
	return &cli.RootCommand{
		Name:    "tagrep",
		Version: version.HumanVersion,
		Commands: map[string]cli.CommandFactory{
			"issue": func() cli.Command {
				return &issue.ParseCommand{}
			},
			"request": func() cli.Command {
				return &request.ParseCommand{}
			},
		},
	}
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer done()

	if err := realMain(ctx); err != nil {
		done()
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func setupMetricsClient(ctx context.Context) context.Context {
	mClient, err := metrics.New(ctx, version.Name, version.Version)
	if err != nil {
		logging.FromContext(ctx).DebugContext(ctx, "metric client creation failed", "error", err)
	}

	ctx = metrics.WithClient(ctx, mClient)
	return ctx
}

func realMain(ctx context.Context) error {
	start := time.Now()
	setLogEnvVars()
	ctx = logging.WithLogger(ctx, logging.NewFromEnv("TAGREP_"))

	ctx = setupMetricsClient(ctx)
	defer func() {
		if r := recover(); r != nil {
			metricswrap.WriteMetric(ctx, "panics", 1)
			panic(r)
		}
	}()

	metricswrap.WriteMetric(ctx, "runs", 1)
	defer func() {
		// Needs to be wrapped in func() due to time.Since(start).
		metricswrap.WriteMetric(ctx, "runtime_millis", time.Since(start).Milliseconds())
	}()

	return rootCmd().Run(ctx, os.Args[1:]) //nolint:wrapcheck // Want passthrough
}

// setLogEnvVars set the logging environment variables to their default
// values if not provided.
func setLogEnvVars() {
	if os.Getenv("TAGREP_LOG_FORMAT") == "" {
		os.Setenv("TAGREP_LOG_FORMAT", defaultLogFormat)
	}

	if os.Getenv("TAGREP_LOG_LEVEL") == "" {
		os.Setenv("TAGREP_LOG_LEVEL", defaultLogLevel)
	}

	if os.Getenv("TAGREP_LOG_DEBUG") == "" {
		os.Setenv("TAGREP_LOG_DEBUG", defaultLogDebug)
	}
}
