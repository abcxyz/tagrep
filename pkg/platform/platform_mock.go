// Copyright 2024 The Authors (see AUTHORS file)
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
	"sync"
)

var _ Platform = (*MockPlatform)(nil)

type Request struct {
	Name   string
	Params []any
}

type MockPlatform struct {
	reqMu sync.Mutex
	Reqs  []*Request

	IsPullRequest bool
	IncludeTeams  bool

	GetRequestBodyErr      error
	GetRequestBodyResponse string
	GetIssueBodyErr        error
	GetIssueBodyResponse   string
}

func (m *MockPlatform) GetRequestBody(ctx context.Context) (string, error) {
	m.reqMu.Lock()
	defer m.reqMu.Unlock()
	m.Reqs = append(m.Reqs, &Request{
		Name:   "GetRequestBody",
		Params: []any{},
	})

	if m.GetRequestBodyErr != nil {
		return "", m.GetRequestBodyErr
	}

	return m.GetRequestBodyResponse, nil
}

func (m *MockPlatform) GetIssueBody(ctx context.Context) (string, error) {
	m.reqMu.Lock()
	defer m.reqMu.Unlock()
	m.Reqs = append(m.Reqs, &Request{
		Name:   "GetIssueBody",
		Params: []any{},
	})

	if m.GetIssueBodyErr != nil {
		return "", m.GetIssueBodyErr
	}

	return m.GetIssueBodyResponse, nil
}
