package state

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/job"
)

type JobStore struct {
	db        *memdb.MemDB
	tableName string
	logger    *log.Logger

	nextJobOpenDirMu   *sync.Mutex
	nextJobClosedDirMu *sync.Mutex

	lastJobId uint64
}

type ScheduledJob struct {
	job.ID
	job.Job
	IsDirOpen bool
	State     State

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

	txn := js.db.Txn(true)
	defer txn.Abort()

	newJobID := job.ID(fmt.Sprintf("%d", atomic.AddUint64(&js.lastJobId, 1)))

	err = txn.Insert(js.tableName, &ScheduledJob{
		ID:        newJobID,
		Job:       newJob,
		IsDirOpen: isDirOpen(txn, newJob.Dir),
		State:     StateQueued,
	})
	if err != nil {
		return "", err
	}

	js.logger.Printf("JOBS: Enqueueing new job %q: %q for %q", newJobID, newJob.Type, newJob.Dir)

	txn.Commit()

	return newJobID, nil
}

func (js *JobStore) DequeueJobsForDir(dir document.DirHandle) error {
	txn := js.db.Txn(true)
	defer txn.Abort()

	js.logger.Printf("JOBS: Dequeueing jobs for %q", dir.URI)

	it, err := txn.Get(jobsTableName, "dir_state", dir, StateQueued)
	if err != nil {
		return fmt.Errorf("failed to find queued jobs for %q: %w", dir, err)
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		sJob := obj.(*ScheduledJob)

		_, err = txn.DeleteAll(js.tableName, "id", sJob.ID)
		if err != nil {
			return err
		}

		err = js.cleanupParentDoneJobsOf(txn, sJob.ID)
		if err != nil {
			return err
		}
	}

	txn.Commit()
	return nil
}

func jobsExistForDirHandle(txn *memdb.Txn, dir document.DirHandle) (<-chan struct{}, bool, error) {
	wCh, runningObj, err := txn.FirstWatch(jobsTableName, "dir_state", dir, StateRunning)
	if err != nil {
		return nil, false, err
	}
	if runningObj != nil {
		return wCh, true, nil
	}

	queuedObj, err := txn.First(jobsTableName, "dir_state", dir, StateQueued)
	if err != nil {
		return nil, false, err
	}
	if queuedObj != nil {
		return wCh, true, nil
	}

	return nil, false, nil
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
		// Although we hold a write db-wide lock when marking job as running
		// we may still end up passing the same job from the above read-only
		// transaction, which does *not* hold a db-wide lock.
		//
		// Instead of adding more sync primitives here we simply retry.
		if errors.Is(err, jobAlreadyRunning{ID: sJob.ID}) || errors.Is(err, jobNotFound{ID: sJob.ID}) {
			js.logger.Printf("retrying next job: %s", err)
			return js.awaitNextJob(ctx, openDir)
		}

		return "", job.Job{}, err
	}

	js.logger.Printf("JOBS: Dispatching next job %q: %q for %q", sJob.ID, sJob.Type, sJob.Dir)
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
			ids, err := js.waitForJobId(ctx, id)
			if err != nil {
				js.logger.Printf("error waiting for job %q: %s", id, err)
				return
			}
			deferredJobIds = append(deferredJobIds, ids...)
		}

		err := js.WaitForJobs(ctx, deferredJobIds...)
		if err != nil {
			js.logger.Printf("error waiting for %d deferred jobs: %s", len(deferredJobIds), err)
		}
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

	if sj.State == StateRunning {
		return jobAlreadyRunning{ID: sJob.ID}
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

	return nil
}

func (js *JobStore) FinishJob(id job.ID, jobErr error, deferredJobIds ...job.ID) error {
	txn := js.db.Txn(true)
	defer txn.Abort()

	sj, err := copyJob(txn, id)
	if err != nil {
		return fmt.Errorf("failed to copy a job: %w", err)
	}

	js.logger.Printf("JOBS: Finishing job %q: %q for %q (err = %s, deferredJobs: %q)",
		sj.ID, sj.Type, sj.Dir, jobErr, deferredJobIds)

	_, err = txn.DeleteAll(js.tableName, "id", id)
	if err != nil {
		return err
	}

	if len(deferredJobIds) == 0 {
		err = js.cleanupParentDoneJobsOf(txn, id)
		if err != nil {
			return err
		}
		txn.Commit()

		return nil
	}

	sj.Func = nil
	sj.State = StateDone
	sj.JobErr = jobErr
	sj.DeferredJobIDs = deferredJobIds

	err = txn.Insert(js.tableName, sj)
	if err != nil {
		return err
	}

	txn.Commit()

	return nil
}

func (js *JobStore) cleanupParentDoneJobsOf(txn *memdb.Txn, id job.ID) error {
	it, err := txn.Get(js.tableName, "state", StateDone)
	if err != nil {
		return err
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		parentJob := obj.(*ScheduledJob)

		if len(parentJob.DeferredJobIDs) == 1 && parentJob.DeferredJobIDs[0] == id {
			// short-circuit if there are no more jobs
			// to avoid unnecessary copying
			_, err = txn.DeleteAll(js.tableName, "id", parentJob.ID)
			if err != nil {
				return err
			}

			err = js.cleanupParentDoneJobsOf(txn, parentJob.ID)
			if err != nil {
				return err
			}

			continue
		}

		i, ok := idIsInSlice(parentJob.DeferredJobIDs, id)
		if !ok {
			continue
		}

		job, err := copyJob(txn, parentJob.ID)
		if err != nil {
			return fmt.Errorf("failed to copy a job %q: %w", parentJob.ID, err)
		}

		_, err = txn.DeleteAll(js.tableName, "id", parentJob.ID)
		if err != nil {
			return err
		}

		// remove ID from the slice
		job.DeferredJobIDs[i] = job.DeferredJobIDs[len(job.DeferredJobIDs)-1]
		job.DeferredJobIDs = job.DeferredJobIDs[:len(job.DeferredJobIDs)-1]

		err = txn.Insert(js.tableName, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func idIsInSlice(ids job.IDs, id job.ID) (int, bool) {
	for i, jobId := range ids {
		if jobId == id {
			return i, true
		}
	}
	return 0, false
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

func (js *JobStore) allJobs() (job.IDs, error) {
	txn := js.db.Txn(false)

	it, err := txn.Get(js.tableName, "id")
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
	return nil, jobNotFound{ID: id}
}
