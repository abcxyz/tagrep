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
	"os"
	"strconv"
	"time"

	"github.com/sethvargo/go-retry"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/abcxyz/pkg/cli"
)

var (
	_ Platform = (*GitLab)(nil)

	// gitLabIgnoredStatusCodes are status codes that should not be retried. This
	// list is taken from the GitLab REST API documentation but may not contain
	// the full set of status codes to ignore.
	// See https://docs.gitlab.com/ee/api/rest/troubleshooting.html#status-codes.
	gitLabIgnoredStatusCodes = map[int]struct{}{
		403: {},
		405: {},
		422: {},
	}
)

// GitLab implements the Platform interface.
type GitLab struct {
	cfg    *gitLabConfig
	client *gitlab.Client
}

type gitLabConfig struct {
	// Retry
	MaxRetries        uint64
	InitialRetryDelay time.Duration
	MaxRetryDelay     time.Duration

	TagrepGitLabToken string
	GitLabBaseURL     string

	GitLabProjectID       int
	GitLabMergeRequestIID int
	GitLabIssueIID        int
}

type gitLabPredefinedConfig struct {
	CIJobToken   string
	CIServerHost string
	CIProjectID  int
	// The merge request IID is the number used in the GitLab API, and not ID.
	// See https://docs.gitlab.com/ee/ci/variables/predefined_variables.html.
	CIMergeRequestIID int
	// The issue IID is the number used in the GitLab API, and not ID.
	// See https://docs.gitlab.com/ee/ci/variables/predefined_variables.html.
	CIIssueIID int
}

// Load retrieves the predefined GitLab CI/CD variables from environment. See
// https://docs.gitlab.com/ee/ci/variables/predefined_variables.html#predefined-variables.
func (c *gitLabPredefinedConfig) Load() {
	if v := os.Getenv("CI_JOB_TOKEN"); v != "" {
		c.CIJobToken = v
	}

	if v := os.Getenv("CI_API_V4_URL"); v != "" {
		c.CIServerHost = v
	}

	if v, err := strconv.Atoi(os.Getenv("CI_PROJECT_ID")); err == nil {
		c.CIProjectID = v
	}

	if v, err := strconv.Atoi(os.Getenv("CI_MERGE_REQUEST_IID")); err == nil {
		c.CIMergeRequestIID = v
	}

	if v, err := strconv.Atoi(os.Getenv("CI_ISSUE_IID")); err == nil {
		c.CIIssueIID = v
	}
}

func (c *gitLabConfig) RegisterFlags(set *cli.FlagSet) {
	f := set.NewSection("GITLAB OPTIONS")

	cfgDefaults := &gitLabPredefinedConfig{}
	cfgDefaults.Load()

	f.StringVar(&cli.StringVar{
		Name:    "tagrep-gitlab-token",
		EnvVar:  "TAGREP_GITLAB_TOKEN",
		Target:  &c.TagrepGitLabToken,
		Default: cfgDefaults.CIJobToken,
		Usage:   "The GitLab access token to make GitLab API calls.",
		Hidden:  true,
	})

	f.StringVar(&cli.StringVar{
		Name:    "gitlab-base-url",
		EnvVar:  "GITLAB_BASE_URL",
		Target:  &c.GitLabBaseURL,
		Example: "https://git.mydomain.com/api/v4",
		Default: cfgDefaults.CIServerHost,
		Usage:   "The base URL of the GitLab instance API.",
		Hidden:  true,
	})

	f.IntVar(&cli.IntVar{
		Name:    "gitlab-project-id",
		EnvVar:  "GITLAB_PROJECT_ID",
		Target:  &c.GitLabProjectID,
		Default: cfgDefaults.CIProjectID,
		Usage:   "The GitLab project ID.",
		Hidden:  true,
	})

	f.IntVar(&cli.IntVar{
		Name:    "gitlab-merge-request-iid",
		EnvVar:  "GITLAB_MERGE_REQUEST_IID",
		Target:  &c.GitLabMergeRequestIID,
		Default: cfgDefaults.CIMergeRequestIID,
		Usage:   "The GitLab project-level merge request internal ID.",
		Hidden:  true,
	})

	f.IntVar(&cli.IntVar{
		Name:    "gitlab-issue-iid",
		EnvVar:  "GITLAB_ISSUE_IID",
		Target:  &c.GitLabIssueIID,
		Default: cfgDefaults.CIMergeRequestIID,
		Usage:   "The GitLab project-level issue internal ID.",
		Hidden:  true,
	})
}

// NewGitLab creates a new GitLab client.
func NewGitLab(ctx context.Context, cfg *gitLabConfig) (*GitLab, error) {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialRetryDelay <= 0 {
		cfg.InitialRetryDelay = 1 * time.Second
	}
	if cfg.MaxRetryDelay <= 0 {
		cfg.MaxRetryDelay = 20 * time.Second
	}

	if cfg.GitLabBaseURL == "" {
		return nil, fmt.Errorf("gitlab base url is required")
	}

	c, err := gitlab.NewClient(cfg.TagrepGitLabToken, gitlab.WithBaseURL(cfg.GitLabBaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}

	return &GitLab{
		client: c,
		cfg:    cfg,
	}, nil
}

// GetRequestBody gets the Merge Request description.
func (g *GitLab) GetRequestBody(ctx context.Context) (string, error) {
	if err := validateGitLabInputs(g.cfg); err != nil {
		return "", fmt.Errorf("failed to validate inputs: %w", err)
	}
	var body string

	if err := g.withRetries(ctx, func(ctx context.Context) error {
		mr, resp, err := g.client.MergeRequests.GetMergeRequest(g.cfg.GitLabProjectID, g.cfg.GitLabMergeRequestIID, nil)
		if err != nil {
			return gitlabMaybeRetryable(resp, fmt.Errorf("failed to get merge request: %w", err))
		}
		body = mr.Description

		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to list reports: %w", err)
	}
	return body, nil
}

// GetIssueBody gets the body of the issue.
func (g *GitLab) GetIssueBody(ctx context.Context) (string, error) {
	if err := validateGitLabInputs(g.cfg); err != nil {
		return "", fmt.Errorf("failed to validate inputs: %w", err)
	}
	var body string

	if err := g.withRetries(ctx, func(ctx context.Context) error {
		mr, resp, err := g.client.Issues.GetIssue(g.cfg.GitLabProjectID, g.cfg.GitLabIssueIID, nil)
		if err != nil {
			return gitlabMaybeRetryable(resp, fmt.Errorf("failed to get issue: %w", err))
		}
		body = mr.Description

		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to list reports: %w", err)
	}
	return body, nil
}

func validateGitLabInputs(cfg *gitLabConfig) error {
	var merr error
	if cfg.GitLabProjectID <= 0 {
		merr = errors.Join(merr, fmt.Errorf("gitlab project id is required"))
	}

	if cfg.GitLabMergeRequestIID <= 0 && cfg.GitLabIssueIID <= 0 {
		merr = errors.Join(merr, fmt.Errorf("gitlab merge request id or issue id is required"))
	}

	if cfg.TagrepGitLabToken == "" {
		merr = errors.Join(merr, fmt.Errorf("gitlab token is required"))
	}

	return merr
}

func (g *GitLab) withRetries(ctx context.Context, retryFunc retry.RetryFunc) error {
	backoff := retry.NewFibonacci(g.cfg.InitialRetryDelay)
	backoff = retry.WithMaxRetries(g.cfg.MaxRetries, backoff)
	backoff = retry.WithCappedDuration(g.cfg.MaxRetryDelay, backoff)

	if err := retry.Do(ctx, backoff, retryFunc); err != nil {
		return fmt.Errorf("failed to execute retriable function: %w", err)
	}
	return nil
}

func gitlabMaybeRetryable(resp *gitlab.Response, err error) error {
	if resp != nil {
		if _, ok := gitLabIgnoredStatusCodes[resp.StatusCode]; !ok {
			return retry.RetryableError(err)
		}
	}
	return err
}
