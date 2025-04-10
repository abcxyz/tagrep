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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/sethvargo/go-githubactions"
	"github.com/sethvargo/go-retry"
	"golang.org/x/oauth2"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/logging"
)

var (
	_ Platform = (*GitHub)(nil)

	// ignoredStatusCodes are status codes that should not be retried. This list
	// is taken from the GitHub REST API documentation.
	ignoredStatusCodes = map[int]struct{}{
		403: {},
		422: {},
	}
)

// GitHub implements the Platform interface.
type GitHub struct {
	cfg    *gitHubConfig
	client *github.Client
}

// mergeGroupPullRequestNumberPattern is a Regex pattern used to parse the pull request number from the merge_group ref.
var mergeGroupPullRequestNumberPattern = regexp.MustCompile(`refs\/heads\/gh-readonly-queue\/(?:.+)\/pr-(\d*)`)

// gitHubConfig is the config values for the GitHub client.
type gitHubConfig struct {
	// Retry
	MaxRetries        uint64
	InitialRetryDelay time.Duration
	MaxRetryDelay     time.Duration

	// Auth
	GitHubToken             string
	GitHubOwner             string
	GitHubRepo              string
	GitHubAppID             string
	GitHubAppInstallationID string
	GitHubAppPrivateKeyPEM  string
	Permissions             map[string]string

	GitHubEventName         string
	GitHubServerURL         string
	GitHubRunID             int64
	GitHubRunAttempt        int64
	GitHubJob               string
	GitHubJobName           string
	GitHubPullRequestNumber int
	GitHubPullRequestBody   string
	GitHubIssueNumber       int
	GitHubIssueBody         string
	GitHubSHA               string
	GitHubActor             string

	configDefaults *gitHubConfigDefaults
}

type gitHubConfigDefaults struct {
	Owner             string
	Repo              string
	PullRequestNumber int
	PullRequestBody   string
	IssueNumber       int
	IssueBody         string
}

// Load retrieves the predefined GitHub CI/CD variables from environment.
func (c *gitHubConfigDefaults) Load(githubContext *githubactions.GitHubContext) {
	c.Owner, c.Repo = githubContext.Repo()
	// we want a typed struct so we will "re-parse" the event payload based on event name.
	// ignore err because we have no way of returning an error via the flags.Register function.
	// this is ok beause this is just for defaulting values from the environment.
	data, _ := json.Marshal(githubContext.Event) //nolint:errchkjson // Shouldnt affect defaults
	switch githubContext.EventName {
	case "pull_request":
		var event github.PullRequestEvent
		if err := json.Unmarshal(data, &event); err == nil {
			c.PullRequestNumber = event.GetNumber()
			c.PullRequestBody = event.GetPullRequest().GetBody()
		} else {
			logging.DefaultLogger().Warn("parsing pull_request event context failed", "error", err) //nolint:sloglint
		}
		break
	case "pull_request_target":
		var event github.PullRequestTargetEvent
		if err := json.Unmarshal(data, &event); err == nil {
			c.PullRequestNumber = event.GetNumber()
			c.PullRequestBody = event.GetPullRequest().GetBody()
		} else {
			logging.DefaultLogger().Warn("parsing pull_request_target event context failed", "error", err) //nolint:sloglint
		}
		break
	case "pull_request_review":
		var event github.PullRequestReviewEvent
		if err := json.Unmarshal(data, &event); err == nil {
			c.PullRequestNumber = event.GetPullRequest().GetNumber()
			c.PullRequestBody = event.GetPullRequest().GetBody()
		} else {
			logging.DefaultLogger().Warn("parsing pull_request_review event context failed", "error", err) //nolint:sloglint
		}
		break
	case "merge_group":
		var event github.MergeGroupEvent
		if err := json.Unmarshal(data, &event); err == nil {
			matches := mergeGroupPullRequestNumberPattern.FindStringSubmatch(event.GetMergeGroup().GetHeadRef())
			if len(matches) == 2 {
				if v, err := strconv.Atoi(matches[1]); err == nil {
					c.PullRequestNumber = v
				} else {
					logging.DefaultLogger().Warn("parsing merge_group head_ref for pull request number failed", //nolint:sloglint
						"head_ref", event.GetMergeGroup().GetHeadRef(),
						"error", err)
				}
			} else {
				logging.DefaultLogger().Warn("parsing merge_group head_ref for pull request number failed", "head_ref", event.GetMergeGroup().GetHeadRef()) //nolint:sloglint
			}
			// Pull request body is not available on the merge_group event.
		} else {
			logging.DefaultLogger().Warn("parsing merge_group event context failed", "error", err) //nolint:sloglint
		}
	case "issues":
		var event github.IssuesEvent
		if err := json.Unmarshal(data, &event); err == nil {
			c.IssueNumber = event.GetIssue().GetNumber()
			c.IssueBody = event.GetIssue().GetBody()
		} else {
			logging.DefaultLogger().Warn("parsing issues event context failed", "error", err) //nolint:sloglint
		}
		break
	default:
		logging.DefaultLogger().Warn("unhandled tagrep github event context type", "context_event_name", githubContext.EventName) //nolint:sloglint
	}
}

func (c *gitHubConfig) RegisterFlags(set *cli.FlagSet) {
	gitHubContext, _ := githubactions.New().Context()
	c.configDefaults = &gitHubConfigDefaults{}
	c.configDefaults.Load(gitHubContext)

	f := set.NewSection("GITHUB OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:   "github-token",
		EnvVar: "GITHUB_TOKEN",
		Target: &c.GitHubToken,
		Usage:  "The GitHub access token to make GitHub API calls.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:    "github-owner",
		Target:  &c.GitHubOwner,
		Default: c.configDefaults.Owner,
		Example: "organization-name",
		Usage:   "The GitHub repository owner.",
		Hidden:  true,
	})

	f.StringVar(&cli.StringVar{
		Name:    "github-repo",
		Target:  &c.GitHubRepo,
		Default: c.configDefaults.Repo,
		Example: "repository-name",
		Usage:   "The GitHub repository name.",
		Hidden:  true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-app-id",
		EnvVar: "GITHUB_APP_ID",
		Target: &c.GitHubAppID,
		Usage:  "The ID of GitHub App to use for requesting tokens to make GitHub API calls.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-app-installation-id",
		EnvVar: "GITHUB_APP_INSTALLATION_ID",
		Target: &c.GitHubAppInstallationID,
		Usage:  "The Installation ID of GitHub App to use for requesting tokens to make GitHub API calls.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-app-private-key-pem",
		EnvVar: "GITHUB_APP_PRIVATE_KEY_PEM",
		Target: &c.GitHubAppPrivateKeyPEM,
		Usage:  "The PEM formatted private key to use with the GitHub App.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-server-url",
		EnvVar: "GITHUB_SERVER_URL",
		Target: &c.GitHubServerURL,
		Usage:  "The GitHub server URL.",
		Hidden: true,
	})

	f.Int64Var(&cli.Int64Var{
		Name:   "github-run-id",
		EnvVar: "GITHUB_RUN_ID",
		Target: &c.GitHubRunID,
		Usage:  "The GitHub workflow run ID.",
		Hidden: true,
	})

	f.Int64Var(&cli.Int64Var{
		Name:   "github-run-attempt",
		EnvVar: "GITHUB_RUN_ATTEMPT",
		Target: &c.GitHubRunAttempt,
		Usage:  "The GitHub workflow run attempt.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-event-name",
		EnvVar: "GITHUB_EVENT_NAME",
		Target: &c.GitHubEventName,
		Usage:  "The GitHub event name.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-job",
		EnvVar: "GITHUB_JOB",
		Target: &c.GitHubJob,
		Usage:  "The GitHub job id.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-job-name",
		EnvVar: "GITHUB_JOB_NAME",
		Target: &c.GitHubJobName,
		Usage:  "The GitHub job name.",
		Hidden: true,
	})

	f.IntVar(&cli.IntVar{
		Name:    "github-pull-request-number",
		EnvVar:  "GITHUB_PULL_REQUEST_NUMBER",
		Target:  &c.GitHubPullRequestNumber,
		Default: c.configDefaults.PullRequestNumber,
		Usage:   "The GitHub pull request number.",
	})

	f.StringVar(&cli.StringVar{
		Name:    "github-pull-request-body",
		EnvVar:  "GITHUB_PULL_REQUEST_BODY",
		Target:  &c.GitHubPullRequestBody,
		Default: c.configDefaults.PullRequestBody,
		Usage:   "The GitHub pull request body.",
		Hidden:  true,
	})

	f.IntVar(&cli.IntVar{
		Name:    "github-issue-number",
		EnvVar:  "GITHUB_ISSUE_NUMBER",
		Target:  &c.GitHubIssueNumber,
		Default: c.configDefaults.IssueNumber,
		Usage:   "The GitHub issue number.",
	})

	f.StringVar(&cli.StringVar{
		Name:    "github-issue-body",
		EnvVar:  "GITHUB_ISSUE_BODY",
		Target:  &c.GitHubIssueBody,
		Default: c.configDefaults.IssueBody,
		Usage:   "The GitHub issue body.",
		Hidden:  true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-commit-sha",
		EnvVar: "GITHUB_SHA",
		Target: &c.GitHubSHA,
		Usage:  "The GitHub SHA.",
		Hidden: true,
	})

	f.StringVar(&cli.StringVar{
		Name:   "github-actor",
		EnvVar: "GITHUB_ACTOR",
		Target: &c.GitHubActor,
		Usage:  "The GitHub Login of the user requesting the workflow.",
		Hidden: true,
	})

	set.AfterParse(func(merr error) error {
		// The PullRequestNumber and PullRequestBody must derive from the same
		// PR - we reset the body value from the default github context if the
		// user provided a custom PR number.
		userProvidedPRNumberOverride := c.configDefaults.PullRequestNumber > 0 && c.configDefaults.PullRequestNumber != c.GitHubPullRequestNumber
		if userProvidedPRNumberOverride && c.configDefaults.PullRequestBody == c.GitHubPullRequestBody {
			c.GitHubPullRequestBody = ""
		}

		// The IssueNumber and IssueBody must derive from the same issue - we
		// reset the body value from the default github context if the user
		// provided a custom PR number.
		userProvidedIssueNumberOverride := c.configDefaults.IssueNumber > 0 && c.configDefaults.IssueNumber != c.GitHubIssueNumber
		if userProvidedIssueNumberOverride && c.configDefaults.IssueBody == c.GitHubIssueBody {
			c.GitHubIssueBody = ""
		}

		return nil
	})
}

// NewGitHub creates a new GitHub client.
func NewGitHub(ctx context.Context, cfg *gitHubConfig) (*GitHub, error) {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialRetryDelay <= 0 {
		cfg.InitialRetryDelay = 1 * time.Second
	}
	if cfg.MaxRetryDelay <= 0 {
		cfg.MaxRetryDelay = 20 * time.Second
	}

	var ts oauth2.TokenSource
	if cfg.GitHubToken != "" {
		ts = oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: cfg.GitHubToken,
		})
	} else {
		signer, err := githubauth.NewPrivateKeySigner(cfg.GitHubAppPrivateKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		app, err := githubauth.NewApp(cfg.GitHubAppID, signer)
		if err != nil {
			return nil, fmt.Errorf("failed to create github app token source: %w", err)
		}

		installation, err := app.InstallationForID(ctx, cfg.GitHubAppInstallationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get github app installation: %w", err)
		}

		ts = installation.SelectedReposOAuth2TokenSource(ctx, cfg.Permissions, cfg.GitHubRepo)
	}

	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	g := &GitHub{
		cfg:    cfg,
		client: client,
	}

	return g, nil
}

// GetRequestBody gets the Pull Request body.
func (g *GitHub) GetRequestBody(ctx context.Context) (string, error) {
	if g.cfg.GitHubPullRequestBody != "" {
		return g.cfg.GitHubPullRequestBody, nil
	}
	if err := validateGitHubInputs(g.cfg); err != nil {
		return "", fmt.Errorf("failed to validate inputs: %w", err)
	}
	var body string

	if err := g.withRetries(ctx, func(ctx context.Context) error {
		ghPullRequest, resp, err := g.client.PullRequests.Get(ctx, g.cfg.GitHubOwner, g.cfg.GitHubRepo, g.cfg.GitHubPullRequestNumber)
		if err != nil {
			return githubMaybeRetryable(resp, fmt.Errorf("failed to get pull request: %w", err))
		}

		if ghPullRequest != nil && ghPullRequest.Body != nil {
			body = *ghPullRequest.Body
		}

		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to get pull request body: %w", err)
	}

	return body, nil
}

// GetIssueBody gets the Issue body.
func (g *GitHub) GetIssueBody(ctx context.Context) (string, error) {
	if g.cfg.GitHubIssueBody != "" {
		return g.cfg.GitHubIssueBody, nil
	}
	if err := validateGitHubInputs(g.cfg); err != nil {
		return "", fmt.Errorf("failed to validate inputs: %w", err)
	}
	var body string

	if err := g.withRetries(ctx, func(ctx context.Context) error {
		ghIssue, resp, err := g.client.Issues.Get(ctx, g.cfg.GitHubOwner, g.cfg.GitHubRepo, g.cfg.GitHubIssueNumber)
		if err != nil {
			return githubMaybeRetryable(resp, fmt.Errorf("failed to get pull request: %w", err))
		}

		body = *ghIssue.Body

		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to get pull request body: %w", err)
	}

	return body, nil
}

func (g *GitHub) withRetries(ctx context.Context, retryFunc retry.RetryFunc) error {
	backoff := retry.NewFibonacci(g.cfg.InitialRetryDelay)
	backoff = retry.WithMaxRetries(g.cfg.MaxRetries, backoff)
	backoff = retry.WithCappedDuration(g.cfg.MaxRetryDelay, backoff)

	if err := retry.Do(ctx, backoff, retryFunc); err != nil {
		return fmt.Errorf("failed to execute retriable function: %w", err)
	}
	return nil
}

// validateGitHubInputs validates the required inputs.
func validateGitHubInputs(cfg *gitHubConfig) error {
	var merr error
	if cfg.GitHubOwner == "" {
		merr = errors.Join(merr, fmt.Errorf("github owner is required"))
	}

	if cfg.GitHubRepo == "" {
		merr = errors.Join(merr, fmt.Errorf("github repo is required"))
	}

	if cfg.GitHubPullRequestNumber <= 0 && cfg.GitHubIssueNumber <= 0 {
		merr = errors.Join(merr, fmt.Errorf("one of github pull request number or github issue number is required"))
	}

	return merr
}

func githubMaybeRetryable(resp *github.Response, err error) error {
	if _, ok := ignoredStatusCodes[resp.StatusCode]; !ok {
		return retry.RetryableError(err)
	}
	return err
}
