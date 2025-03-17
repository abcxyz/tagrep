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

// Package platform defines interfaces for interacting with code review
// platforms.
package platform

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const (
	TypeUnspecified = ""
	TypeGitHub      = "github"
	TypeGitLab      = "gitlab"
)

var (
	allowedTypes = map[string]struct{}{
		TypeGitHub: {},
		TypeGitLab: {},
	}
	// SortedTypes are the sorted Platform types for printing messages and prediction.
	SortedTypes = func() []string {
		allowed := append([]string{}, TypeGitHub, TypeGitLab)
		sort.Strings(allowed)
		return allowed
	}()

	_ Platform = (*GitHub)(nil)
)

// Platform defines the minimum interface for a code review platform.
type Platform interface {
	// GetRequestBody gets the Pull Request or Merge Request body.
	GetRequestBody(ctx context.Context) (string, error)

	// GetIssueBody gets the body of the issue.
	GetIssueBody(ctx context.Context) (string, error)
}

// NewPlatform creates a new platform based on the provided type.
func NewPlatform(ctx context.Context, cfg *Config) (Platform, error) {
	if strings.EqualFold(cfg.Type, TypeGitHub) {
		gc, err := NewGitHub(ctx, &cfg.GitHub)
		if err != nil {
			return nil, fmt.Errorf("failed to create github: %w", err)
		}
		return gc, nil
	}

	if strings.EqualFold(cfg.Type, TypeGitLab) {
		gl, err := NewGitLab(ctx, &cfg.GitLab)
		if err != nil {
			return nil, fmt.Errorf("failed to create gitlab: %w", err)
		}
		return gl, nil
	}
	return nil, fmt.Errorf("unknown platform type: %s", cfg.Type)
}
