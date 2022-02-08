package state

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
)

type JobStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	nextJobOpenDirMu   *sync.Mutex
	nextJobClosedDirMu *sync.Mutex

	ProgressStartHook JobProgressHook
	ProgressHook      JobProgressHook
	ProgressEndHook   JobProgressHook

	progressToken atomic.Value
	jobCount      jobCount
}

func newJobCount() jobCount {
	var queued, running, done int64
	return jobCount{
		StateQueued:  &queued,
		StateRunning: &running,
		StateDone:    &done,
	}
}

type JobProgressHook func(ctx context.Context, pTkn string, dir document.DirHandle, jobType string, jobCount JobCountSnapshot)

func noopProgressHook(context.Context, string, document.DirHandle, string, JobCountSnapshot) {}

type ScheduledJob struct {
	job.ID
	job.Job
	IsDirOpen bool
	State     State

	// TODO: ProgressToken ? (passed within EnqueueJob via ctx by e.g. walker)

	// JobErr contains error when job finishes (State = StateDone)
	JobErr error
	// DeferredJobIDs contains IDs of any deferred jobs
	// set when job finishes (State = StateDone)
	DeferredJobIDs job.IDs
}

func (sj *ScheduledJob) Copy() *ScheduledJob {
	return &ScheduledJob{
		ID:             sj.ID,
		Job:            sj.Job.Copy(),
		IsDirOpen:      sj.IsDirOpen,
		State:          sj.State,
		JobErr:         sj.JobErr,
		DeferredJobIDs: sj.DeferredJobIDs.Copy(),
	}
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=State -output=jobs_state_string.go
type State uint

const (
	StateQueued State = iota
	StateRunning
	StateDone
)

func (js *JobStore) EnqueueJob(newJob job.Job) (job.ID, error) {
	jobID, queued, err := js.jobExists(newJob, StateQueued)
	if err != nil {
		return "", err
	}
	if queued {
		return jobID, nil
	}

	jobID, running, err := js.jobExists(newJob, StateRunning)
	if err != nil {
		return "", err
	}
	if running {
		return jobID, nil
	}

	newID, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}
	newJobID := job.ID(newID)

	txn := js.db.Txn(true)
	defer txn.Abort()

	err = txn.Insert(js.tableName, &ScheduledJob{
		ID:        newJobID,
		Job:       newJob,
		IsDirOpen: isDirOpen(txn, newJob.Dir),
		State:     StateQueued,
	})
	if err != nil {
		return "", err
	}

	js.logger.Printf("JOBS: Enqueueing new job: %q for %q", newJob.Type, newJob.Dir)

	txn.Commit()

	js.jobCount.add(StateQueued, 1)
	progressToken := js.progressToken.Load()
	if progressToken == "" {
		progressToken, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
		js.progressToken.Store(progressToken)
	}

	return newJobID, nil
}

func (js *JobStore) DequeueJobsForDir(dir document.DirHandle) error {
	txn := js.db.Txn(true)
	defer txn.Abort()

	it, err := txn.Get(jobsTableName, "dir_state", dir, StateQueued)
	if err != nil {
		return fmt.Errorf("failed to find queued jobs for %q: %w", dir, err)
	}

	countDelta := jobCountDelta{
		StateQueued:  0,
		StateRunning: 0,
		StateDone:    0,
	}
	for obj := it.Next(); obj != nil; obj = it.Next() {
		sJob := obj.(*ScheduledJob)

		sj, err := copyJob(txn, sJob.ID)
		if err != nil {
			return err
		}

		_, err = txn.DeleteAll(js.tableName, "id", sJob.ID)
		if err != nil {
			return err
		}

		countDelta[sj.State]--
		countDelta[StateDone]++

		sj.State = StateDone
		sj.Defer = nil
		sj.Func = nil
		sj.JobErr = fmt.Errorf("job dequeued")

		err = txn.Insert(jobsTableName, sj)
		if err != nil {
			return err
		}
	}

	txn.Commit()

	ctx := context.Background()
	jobCount := js.jobCount.applyDelta(countDelta)
	progressToken := js.progressToken.Load().(string)
	if progressToken != "" {
		js.ProgressEndHook(ctx, progressToken, dir, "", jobCount)
		js.progressToken.Store("")
		err := js.removeDoneJobs()
		if err != nil {
			return err
		}
	} else {
		js.ProgressHook(ctx, progressToken, dir, "", jobCount)
	}

	return nil
}

func updateJobsDirOpenMark(txn *memdb.Txn, dirHandle document.DirHandle, isDirOpen bool) error {
	it, err := txn.Get(jobsTableName, "dir_state", dirHandle, StateQueued)
	if err != nil {
		return fmt.Errorf("failed to find queued jobs for %q: %w", dirHandle, err)
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		sJob := obj.(*ScheduledJob)

		sj, err := copyJob(txn, sJob.ID)
		if err != nil {
			return err
		}

		_, err = txn.DeleteAll(jobsTableName, "id", sJob.ID)
		if err != nil {
			return err
		}

		sj.IsDirOpen = isDirOpen

		err = txn.Insert(jobsTableName, sj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (js *JobStore) jobExists(j job.Job, state State) (job.ID, bool, error) {
	txn := js.db.Txn(false)

	obj, err := txn.First(js.tableName, "dir_state_type", j.Dir, state, j.Type)
	if err != nil {
		return "", false, err
	}
	if obj != nil {
		sj := obj.(*ScheduledJob)
		return sj.ID, true, nil
	}

	return "", false, nil
}

func (js *JobStore) AwaitNextJob(ctx context.Context, openDir bool) (job.ID, job.Job, error) {
	// Locking is needed if same query is executed in multiple threads,
	// i.e. this method is called at the same time from different threads, at
	// which point txn.FirstWatch would return the same job to more than
	// one thread and we would then end up executing it more than once.
	if openDir {
		js.nextJobOpenDirMu.Lock()
		defer js.nextJobOpenDirMu.Unlock()
	} else {
		js.nextJobClosedDirMu.Lock()
		defer js.nextJobClosedDirMu.Unlock()
	}

	return js.awaitNextJob(ctx, openDir)
}

func (js *JobStore) awaitNextJob(ctx context.Context, openDir bool) (job.ID, job.Job, error) {
	txn := js.db.Txn(false)

	wCh, obj, err := txn.FirstWatch(js.tableName, "is_dir_open_state", openDir, StateQueued)
	if err != nil {
		return "", job.Job{}, err
	}

	if obj == nil {
		select {
		case <-wCh:
		case <-ctx.Done():
			return "", job.Job{}, ctx.Err()
		}

		return js.awaitNextJob(ctx, openDir)
	}

	sJob := obj.(*ScheduledJob)

	err = js.markJobAsRunning(sJob)
	if err != nil {
		return "", job.Job{}, err
	}

	js.logger.Printf("JOBS: Dispatching next job: %q for %q", sJob.Type, sJob.Dir)
	return sJob.ID, sJob.Job, nil
}

func isDirOpen(txn *memdb.Txn, dirHandle document.DirHandle) bool {
	docObj, err := txn.First(documentsTableName, "id", dirHandle)
	if err != nil {
		return false
	}

	return docObj != nil
}

func (js *JobStore) WaitForJobs(ctx context.Context, ids ...job.ID) error {
	if len(ids) == 0 {
		return nil
	}

	doneCh := make(chan struct{})
	go func() {
		defer func() {
			close(doneCh)
		}()

		deferredJobIds := make(job.IDs, 0)
		for _, id := range ids {
			js.logger.Printf("waiting for %q", id)
			ids, err := js.waitForJobId(ctx, id)
			if err != nil {
				return
			}
			js.logger.Printf("finished waiting for %q", id)
			deferredJobIds = append(deferredJobIds, ids...)
		}

		js.WaitForJobs(ctx, deferredJobIds...)
	}()

	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (js *JobStore) waitForJobId(ctx context.Context, id job.ID) (job.IDs, error) {
	txn := js.db.Txn(false)

	wCh, obj, err := txn.FirstWatch(js.tableName, "id", id)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, nil
	}

	sJob := obj.(*ScheduledJob)
	if sJob.State != StateDone {
		select {
		case <-wCh:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		return js.waitForJobId(ctx, id)
	}

	return sJob.DeferredJobIDs, nil
}

func (js *JobStore) markJobAsRunning(sJob *ScheduledJob) error {
	txn := js.db.Txn(true)
	defer txn.Abort()

	sj, err := copyJob(txn, sJob.ID)
	if err != nil {
		return err
	}

	_, err = txn.DeleteAll(js.tableName, "id", sJob.ID)
	if err != nil {
		return err
	}

	sj.State = StateRunning

	err = txn.Insert(js.tableName, sj)
	if err != nil {
		return err
	}

	txn.Commit()

	js.jobCount.add(StateRunning, 1)
	jobCount := js.jobCount.add(StateQueued, -1)
	ctx := context.Background()
	progressToken := js.progressToken.Load().(string)
	if jobCount[StateRunning] == 1 && jobCount[StateDone] == 0 && progressToken != "" {
		js.ProgressStartHook(ctx, progressToken, sJob.Dir, sJob.Type, jobCount)
	} else {
		js.ProgressHook(ctx, progressToken, sJob.Dir, sJob.Type, jobCount)
	}

	return nil
}

func (js *JobStore) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	txn := js.db.Txn(true)
	defer txn.Abort()

	sj, err := copyJob(txn, id)
	if err != nil {
		return err
	}

	js.logger.Printf("JOBS: Finishing job %q: %q for %q (err = %#v)", sj.ID, sj.Type, sj.Dir, jobErr)

	_, err = txn.DeleteAll(js.tableName, "id", id)
	if err != nil {
		return err
	}

	js.jobCount.add(sj.State, -1)
	jobCount := js.jobCount.add(StateDone, 1)

	sj.Func = nil
	sj.State = StateDone
	sj.JobErr = jobErr
	sj.DeferredJobIDs = deferredJobIds

	err = txn.Insert(js.tableName, sj)
	if err != nil {
		return err
	}

	txn.Commit()

	ctx := context.Background()
	progressToken := js.progressToken.Load().(string)

	if jobCount[StateQueued] == 0 && jobCount[StateRunning] == 0 && progressToken != "" {
		js.ProgressEndHook(ctx, progressToken, sj.Dir, sj.Type, jobCount)
		js.progressToken.Store("")
		js.logger.Println("removing done jobs ...")
		err = js.removeDoneJobs()
		if err != nil {
			return err
		}
		js.logger.Println("removed done jobs ...")
	} else {
		js.ProgressHook(ctx, progressToken, sj.Dir, sj.Type, jobCount)
	}

	return nil
}

func (js *JobStore) removeDoneJobs() error {
	txn := js.db.Txn(true)
	defer txn.Abort()

	it, err := txn.Get(js.tableName, "state", StateDone)
	if err != nil {
		return err
	}
	ids := make(job.IDs, 0)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		job := obj.(*ScheduledJob)
		ids = append(ids, job.ID)
		_, err = txn.DeleteAll(js.tableName, "id", job.ID)
		if err != nil {
			return err
		}
	}
	txn.Commit()
	js.jobCount.add(StateDone, int64(-len(ids)))

	return nil
}

func (js *JobStore) ListQueuedJobs() (job.IDs, error) {
	txn := js.db.Txn(false)

	it, err := txn.Get(js.tableName, "state", StateQueued)
	if err != nil {
		return nil, err
	}

	jobIDs := make(job.IDs, 0)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		sj := obj.(*ScheduledJob)
		jobIDs = append(jobIDs, sj.ID)
	}

	return jobIDs, nil
}

func copyJob(txn *memdb.Txn, id job.ID) (*ScheduledJob, error) {
	obj, err := txn.First(jobsTableName, "id", id)
	if err != nil {
		return nil, err
	}
	if obj != nil {
		sj := obj.(*ScheduledJob)
		return sj.Copy(), nil
	}
	return nil, fmt.Errorf("%q: job not found", id)
}
