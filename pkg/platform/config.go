// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/posener/complete/v2"

	"github.com/abcxyz/pkg/cli"
)

// Config is the configuration needed to generate different Platform types.
type Config struct {
	Type string

	GitHub gitHubConfig
	GitLab gitLabConfig
}

func (c *Config) RegisterFlagsContext(ctx context.Context, set *cli.FlagSet) {
	f := set.NewSection("PLATFORM OPTIONS")
	// Type value is loaded in the following order:
	//
	// 1. Explicit value set through --platform flag
	// 2. Inferred environment from well-known environment variables
	// 3. Default value of "local"
	f.StringVar(&cli.StringVar{
		Name:    "platform",
		Target:  &c.Type,
		Example: "github",
		Usage:   fmt.Sprintf("The code review platform for tagrep to integrate with. Allowed values are %q.", SortedTypes),
		Predict: complete.PredictFunc(func(prefix string) []string {
			return SortedTypes
		}),
	})

	// leave last to put help under platform options
	c.GitHub.RegisterFlagsContext(ctx, set)
	c.GitLab.RegisterFlagsContext(ctx, set)

	set.AfterParse(func(merr error) error {
		c.Type = strings.ToLower(strings.TrimSpace(c.Type))

		if _, ok := allowedTypes[c.Type]; !ok && c.Type != TypeUnspecified {
			merr = errors.Join(merr, fmt.Errorf("unsupported value for platform flag: %s", c.Type))
		}

		if c.Type == TypeUnspecified {
			if v, _ := strconv.ParseBool(set.GetEnv("GITHUB_ACTIONS")); v {
				c.Type = TypeGitHub
			}
			if v, _ := strconv.ParseBool(set.GetEnv("GITLAB_CI")); v {
				c.Type = TypeGitLab
			}
		}

		return merr
	})
}
