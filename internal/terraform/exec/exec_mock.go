package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"time"
)

type Mock struct {
	Args          []string      `json:"args"`
	Stdout        string        `json:"stdout"`
	Stderr        string        `json:"stderr"`
	SleepDuration time.Duration `json:"sleep"`
	ExitCode      int           `json:"exit_code"`
}

func (m *Mock) MarshalJSON() ([]byte, error) {
	type t Mock
	return json.Marshal((*t)(m))
}

func (m *Mock) UnmarshalJSON(b []byte) error {
	type t Mock
	return json.Unmarshal(b, (*t)(m))
}

func MockExecutor(m *Mock) *Executor {
	if m == nil {
		m = &Mock{}
	}

	path, ctxFunc := mockCommandCtxFunc(m)
	executor := NewExecutor(context.Background(), path)
	executor.cmdCtxFunc = ctxFunc
	return executor
}

func mockCommandCtxFunc(e *Mock) (string, cmdCtxFunc) {
	return os.Args[0], func(ctx context.Context, path string, arg ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], os.Args[1:]...)

		expectedJson, _ := e.MarshalJSON()
		cmd.Env = []string{"TF_LS_MOCK=" + string(expectedJson)}

		return cmd
	}
}

func ExecuteMock(rawMockData string) int {
	e := &Mock{}
	err := e.UnmarshalJSON([]byte(rawMockData))
	if err != nil {
		fmt.Fprint(os.Stderr, "unable to unmarshal mock response")
		return 1
	}

	givenArgs := os.Args[1:]
	if !reflect.DeepEqual(e.Args, givenArgs) {
		fmt.Fprintf(os.Stderr, "arguments don't match.\nexpected: %q\ngiven: %q\n",
			e.Args, givenArgs)
		return 1
	}

	if e.SleepDuration > 0 {
		time.Sleep(e.SleepDuration)
	}

	fmt.Fprint(os.Stdout, e.Stdout)
	fmt.Fprint(os.Stderr, e.Stderr)

	return e.ExitCode
}
