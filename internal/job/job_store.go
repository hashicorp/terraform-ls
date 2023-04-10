// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
)

type JobStore interface {
	EnqueueJob(newJob Job) (ID, error)
	WaitForJobs(ctx context.Context, ids ...ID) error
}
