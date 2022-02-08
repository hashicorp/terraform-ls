package state

import "sync/atomic"

type jobCount map[State]*int64

type JobCountSnapshot map[State]int64

type jobCountDelta map[State]int64

func (js jobCount) add(s State, delta int64) JobCountSnapshot {
	snapshot := make(JobCountSnapshot)

	snapshot[s] = atomic.AddInt64(js[s], delta)

	if _, ok := snapshot[StateQueued]; !ok {
		snapshot[StateQueued] = atomic.LoadInt64(js[StateQueued])
	}
	if _, ok := snapshot[StateRunning]; !ok {
		snapshot[StateRunning] = atomic.LoadInt64(js[StateRunning])
	}
	if _, ok := snapshot[StateDone]; !ok {
		snapshot[StateDone] = atomic.LoadInt64(js[StateDone])
	}

	return snapshot
}

func (js jobCount) applyDelta(delta jobCountDelta) JobCountSnapshot {
	snapshot := make(JobCountSnapshot)

	if queued, ok := delta[StateQueued]; ok && queued != 0 {
		newCount := atomic.AddInt64(js[StateQueued], queued)
		snapshot[StateQueued] = newCount
	} else {
		snapshot[StateQueued] = js.State(StateQueued)
	}

	if running, ok := delta[StateRunning]; ok && running != 0 {
		newCount := atomic.AddInt64(js[StateRunning], running)
		snapshot[StateRunning] = newCount
	} else {
		snapshot[StateRunning] = js.State(StateRunning)
	}

	if done, ok := delta[StateDone]; ok && done != 0 {
		newCount := atomic.AddInt64(js[StateDone], done)
		snapshot[StateDone] = newCount
	} else {
		snapshot[StateDone] = js.State(StateDone)
	}

	return snapshot
}

func (js jobCount) State(s State) int64 {
	return atomic.LoadInt64(js[s])
}
