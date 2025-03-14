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

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/abcxyz/pkg/cli"
)

var _ Platform = (*GitLab)(nil)

// GitLab implements the Platform interface.
type GitLab struct {
	cfg    *gitLabConfig
	client *gitlab.Client
}

type gitLabConfig struct{}

func (c *gitLabConfig) RegisterFlags(set *cli.FlagSet) {}

// NewGitLab creates a new GitLab client.
func NewGitLab(ctx context.Context, cfg *gitLabConfig) (*GitLab, error) {
	return &GitLab{
		client: nil,
		cfg:    cfg,
	}, nil
}

// GetRequestBody gets the Merge Request body.
func (g *GitLab) GetRequestBody(ctx context.Context) (string, error) {
	return "", nil
}

// GetIssueBody gets the Issue body.
func (g *GitLab) GetIssueBody(ctx context.Context) (string, error) {
	return "", nil
}
