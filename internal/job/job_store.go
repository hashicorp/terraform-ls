// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
)

type JobStore interface {
	EnqueueJob(ctx context.Context, newJob Job) (ID, error)
	WaitForJobs(ctx context.Context, ids ...ID) error
}
