// Copyright 2023 The Authors (see AUTHORS file)
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

	"github.com/google/go-github/v53/github"

	"github.com/abcxyz/pkg/cli"
)

var _ Platform = (*GitHub)(nil)

// GitHub implements the Platform interface.
type GitHub struct {
	cfg    *gitHubConfig
	client *github.Client
}

// gitHubConfig is the config values for the GitHub client.
type gitHubConfig struct{}

func (c *gitHubConfig) RegisterFlags(set *cli.FlagSet) {}

// NewGitHub creates a new GitHub client.
func NewGitHub(ctx context.Context, cfg *gitHubConfig) (*GitHub, error) {
	g := &GitHub{
		cfg:    cfg,
		client: nil,
	}

	return g, nil
}

// GetRequestBody gets the Pull Request body.
func (g *GitHub) GetRequestBody(ctx context.Context) (string, error) {
	return "", nil
}

// GetIssueBody gets the Issue body.
func (g *GitHub) GetIssueBody(ctx context.Context) (string, error) {
	return "", nil
}
