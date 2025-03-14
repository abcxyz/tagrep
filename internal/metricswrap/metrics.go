// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// metricswrap handles writing metrics.
package metricswrap

import (
	"context"
	"time"

	"github.com/abcxyz/abc-updater/pkg/metrics"
	"github.com/abcxyz/pkg/logging"
)

const defaultMetricsTimeout = 500 * time.Millisecond

// WriteMetric is a sync wrapper for metrics.WriteMetric.
// It handles client retrival, timeouts, and errors.
func WriteMetric(ctx context.Context, name string, count int64) {
	client := metrics.FromContext(ctx)
	ctx, done := context.WithTimeout(ctx, defaultMetricsTimeout)
	defer done()
	if err := client.WriteMetric(ctx, name, count); err != nil {
		logging.FromContext(ctx).DebugContext(ctx, "failed to write metric", "err", err)
	}
}
