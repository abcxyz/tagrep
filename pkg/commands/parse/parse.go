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

// Package parse parses the github pull request gitlab merge request, or github/gitlab issue and prints any tags to stdout.
package parse

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/posener/complete/v2"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/tagrep/internal/metricswrap"
	"github.com/abcxyz/tagrep/pkg/platform"
	"github.com/abcxyz/tagrep/pkg/tags"
)

var _ cli.Command = (*ParseCommand)(nil)

const (
	TypeUnspecified = ""
	TypeIssue       = "issue"
	TypeRequest     = "request"
)

var (
	allowedTypes = map[string]struct{}{
		TypeIssue:   {},
		TypeRequest: {},
	}
	sortedTypes = func() []string {
		allowed := append([]string{}, TypeIssue, TypeRequest)
		sort.Strings(allowed)
		return allowed
	}()
)

// ParseCommand fetches and parses a request and prints out all tags.
type ParseCommand struct {
	cli.BaseCommand

	platformConfig platform.Config
	tagsConfig     tags.Config

	platformClient platform.Platform
	tagParser      tags.TagParser

	FlagType string
}

// Desc provides a short, one-line description of the command.
func (c *ParseCommand) Desc() string {
	return "Parse tags and output to stdout"
}

// Help is the long-form help output to include usage instructions and flag
// information.
func (c *ParseCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

	Parse tags and output to stdout. Tags should be of the form:

	TAG_1=Some tag value
	TAG_2=my-tag
	TAG_3=something
	TAG_3=something else
`
}

func (c *ParseCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()

	c.platformConfig.RegisterFlags(set)
	c.tagsConfig.RegisterFlags(set)

	f := set.NewSection("TAGREP OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "type",
		Target:  &c.FlagType,
		Example: "issue",
		Usage:   fmt.Sprintf("Type of version control platform asset to process. Allowed values are %q.", sortedTypes),
		Predict: complete.PredictFunc(func(prefix string) []string {
			return sortedTypes
		}),
	})

	set.AfterParse(func(merr error) error {
		c.FlagType = strings.ToLower(strings.TrimSpace(c.FlagType))

		if _, ok := allowedTypes[c.FlagType]; !ok || c.FlagType == TypeUnspecified {
			merr = errors.Join(merr, fmt.Errorf("unsupported value for type flag: %s", c.FlagType))
		}

		return merr
	})

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
	c.tagParser = tags.NewTagParser(ctx, &c.tagsConfig)

	return c.Process(ctx)
}

// Process handles the main logic for the tagrep request.
func (c *ParseCommand) Process(ctx context.Context) (merr error) {
	logger := logging.FromContext(ctx)
	logger.DebugContext(ctx, "starting tagrep request",
		"platform", c.platformConfig.Type)

	var err error
	var body string
	switch c.FlagType {
	case TypeRequest:
		if body, err = c.platformClient.GetRequestBody(ctx); err != nil {
			return fmt.Errorf("failed to get request body: %w", err)
		}
	case TypeIssue:
		if body, err = c.platformClient.GetIssueBody(ctx); err != nil {
			return fmt.Errorf("failed to get issue body: %w", err)
		}
	default:
		return fmt.Errorf("failed to process tags for unsupported version control object of type %s", c.FlagType)
	}
	ts, err := c.tagParser.ParseTags(ctx, body)
	if err != nil {
		return errors.Join(merr, fmt.Errorf("failed to parse tags: %w", err))
	}

	logger.DebugContext(ctx, "parsed tags from request body",
		"tags", ts)

	c.Outf(ts)

	return merr
}
