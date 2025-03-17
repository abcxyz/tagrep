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

// Package request parses the github pull request or gitlab merge request and exports any tags to env.
package request

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/tagrep/internal/metricswrap"
	"github.com/abcxyz/tagrep/pkg/flags"
	"github.com/abcxyz/tagrep/pkg/platform"
	"github.com/abcxyz/tagrep/pkg/tags"
)

var _ cli.Command = (*ParseCommand)(nil)

// ParseCommand fetches and parses a request and prints out all tags.
type ParseCommand struct {
	cli.BaseCommand

	platformConfig platform.Config

	flags.CommonFlags

	platformClient platform.Platform
}

// Desc provides a short, one-line description of the command.
func (c *ParseCommand) Desc() string {
	return "Export request tags to env."
}

// Help is the long-form help output to include usage instructions and flag
// information.
func (c *ParseCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

	Export request tags to env.
`
}

func (c *ParseCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()

	c.platformConfig.RegisterFlags(set)
	c.CommonFlags.Register(set)

	return set
}

func (c *ParseCommand) Run(ctx context.Context, args []string) error {
	metricswrap.WriteMetric(ctx, "command_request", 1)

	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	parsedArgs := f.Args()
	if len(parsedArgs) > 0 {
		return flag.ErrHelp
	}

	platform, err := platform.NewPlatform(ctx, &c.platformConfig)
	if err != nil {
		return fmt.Errorf("failed to create platform: %w", err)
	}
	c.platformClient = platform

	return c.Process(ctx)
}

// Process handles the main logic for the tagrep request.
func (c *ParseCommand) Process(ctx context.Context) (merr error) {
	logger := logging.FromContext(ctx)
	logger.DebugContext(ctx, "starting tagrep request",
		"platform", c.platformConfig.Type)

	body, err := c.platformClient.GetRequestBody(ctx)
	if err != nil {
		merr = errors.Join(merr, fmt.Errorf("failed to get request body: %w", err))
	}
	ts := tags.ParseTags(body)

	logger.DebugContext(ctx, "parsed tags from request body",
		"tags", ts)

	for _, t := range ts {
		c.Stdout().Write([]byte(fmt.Sprintf("%s=%s", t.Name, t.Value)))
	}

	return merr
}
