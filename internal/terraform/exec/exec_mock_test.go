package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

type expected struct {
	Args          []string      `json:"args"`
	Stdout        string        `json:"stdout"`
	Stderr        string        `json:"stderr"`
	SleepDuration time.Duration `json:"sleep"`
	ExitCode      int           `json:"exit_code"`
}

func (e *expected) MarshalJSON() ([]byte, error) {
	type t expected
	return json.Marshal((*t)(e))
}

func (e *expected) UnmarshalJSON(b []byte) error {
	type t expected
	return json.Unmarshal(b, (*t)(e))
}

func mockCommandCtxFunc(e *expected) cmdCtxFunc {
	return func(ctx context.Context, path string, arg ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], os.Args[1:]...)

		expectedJson, _ := e.MarshalJSON()
		cmd.Env = []string{"TF_LS_MOCK=" + string(expectedJson)}

		return cmd
	}
}

func TestMain(m *testing.M) {
	if v := os.Getenv("TF_LS_MOCK"); v != "" {
		e := &expected{}
		err := e.UnmarshalJSON([]byte(v))
		if err != nil {
			fmt.Fprint(os.Stderr, "unable to unmarshal mock response")
			os.Exit(1)
		}

		givenArgs := os.Args[1:]
		if !reflect.DeepEqual(e.Args, givenArgs) {
			fmt.Fprintf(os.Stderr, "arguments don't match.\nexpected: %q\ngiven: %q\n",
				e.Args, givenArgs)
			os.Exit(1)
		}

		if e.SleepDuration > 0 {
			time.Sleep(e.SleepDuration)
		}

		fmt.Fprint(os.Stdout, e.Stdout)
		fmt.Fprint(os.Stderr, e.Stderr)

		os.Exit(e.ExitCode)
		return
	}

	os.Exit(m.Run())
}
