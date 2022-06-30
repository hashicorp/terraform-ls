package scheduler

import (
	"context"
	"errors"
	"io/ioutil"
	"log"

	"github.com/hashicorp/terraform-ls/internal/job"
)

type Scheduler struct {
	logger      *log.Logger
	jobStorage  JobStorage
	parallelism int
	priority    job.JobPriority
	stopFunc    context.CancelFunc
}

type JobStorage interface {
	job.JobStore
	AwaitNextJob(ctx context.Context, priority job.JobPriority) (job.ID, job.Job, error)
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
		id, nextJob, err := s.jobStorage.AwaitNextJob(ctx, s.priority)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Printf("failed to obtain next job: %s", err)
			return
		}

		jobErr := nextJob.Func(ctx)

		deferredJobIds := make(job.IDs, 0)
		if nextJob.Defer != nil {
			deferCtx := job.WithJobStore(ctx, s.jobStorage)
			deferredJobIds = nextJob.Defer(deferCtx, jobErr)
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
