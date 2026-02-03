// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package scheduler

import (
	"context"
	"errors"
	"io/ioutil"
	"log"

	"github.com/hashicorp/terraform-ls/internal/job"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/hashicorp/terraform-ls/internal/scheduler"

type Scheduler struct {
	logger      *log.Logger
	jobStorage  JobStorage
	parallelism int
	priority    job.JobPriority
	stopFunc    context.CancelFunc
}

type JobStorage interface {
	job.JobStore
	AwaitNextJob(ctx context.Context, priority job.JobPriority) (context.Context, job.ID, job.Job, error)
	FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error
}

func NewScheduler(jobStorage JobStorage, parallelism int, priority job.JobPriority) *Scheduler {
	discardLogger := log.New(ioutil.Discard, "", 0)

	return &Scheduler{
		logger:      discardLogger,
		jobStorage:  jobStorage,
		parallelism: parallelism,
		priority:    priority,
		stopFunc:    func() {},
	}
}

func (s *Scheduler) SetLogger(logger *log.Logger) {
	s.logger = logger
}

func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	s.stopFunc = cancelFunc

	for i := 0; i < s.parallelism; i++ {
		s.logger.Printf("launching eval loop %d", i)
		go s.eval(ctx)
	}
}

func (s *Scheduler) Stop() {
	s.stopFunc()
	s.logger.Print("stopped scheduler")
}

func (s *Scheduler) eval(ctx context.Context) {
	for {
		ctx, id, nextJob, err := s.jobStorage.AwaitNextJob(ctx, s.priority)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Printf("failed to obtain next job: %s", err)
			return
		}

		ctx = job.WithIgnoreState(ctx, nextJob.IgnoreState)
		jobSpan := trace.SpanFromContext(ctx)

		ctx, span := otel.Tracer(tracerName).Start(ctx, "job-eval:"+nextJob.Type,
			trace.WithAttributes(attribute.KeyValue{
				Key:   attribute.Key("JobID"),
				Value: attribute.StringValue(id.String()),
			}, attribute.KeyValue{
				Key:   attribute.Key("JobType"),
				Value: attribute.StringValue(nextJob.Type),
			}, attribute.KeyValue{
				Key:   attribute.Key("Priority"),
				Value: attribute.IntValue(int(nextJob.Priority)),
			}, attribute.KeyValue{
				Key:   attribute.Key("URI"),
				Value: attribute.StringValue(nextJob.Dir.URI),
			}))

		jobErr := nextJob.Func(ctx)

		if jobErr != nil {
			if errors.Is(jobErr, job.StateNotChangedErr{Dir: nextJob.Dir}) {
				span.SetStatus(codes.Ok, "state not changed")
			} else {
				span.RecordError(jobErr)
				span.SetStatus(codes.Error, "job failed")
			}
		} else {
			span.SetStatus(codes.Ok, "job finished")
		}
		span.End()
		jobSpan.SetStatus(codes.Ok, "ok")
		jobSpan.End()

		deferredJobIds := make(job.IDs, 0)
		if nextJob.Defer != nil {
			deferredJobIds, err = nextJob.Defer(ctx, jobErr)
			if err != nil {
				s.logger.Printf("deferred job failed: %s", err)
			}
		}

		err = s.jobStorage.FinishJob(id, jobErr, deferredJobIds...)
		if err != nil {
			s.logger.Printf("failed to finish job: %s", err)
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
