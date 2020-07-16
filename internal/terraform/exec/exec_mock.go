package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"time"
)

type MockItemDispenser interface {
	NextMockItem() *MockItem
}

type MockItem struct {
	Args          []string      `json:"args"`
	Stdout        string        `json:"stdout"`
	Stderr        string        `json:"stderr"`
	SleepDuration time.Duration `json:"sleep"`
	ExitCode      int           `json:"exit_code"`

	MockError string `json:"error"`
}

func (m *MockItem) MarshalJSON() ([]byte, error) {
	type t MockItem
	return json.Marshal((*t)(m))
}

func (m *MockItem) UnmarshalJSON(b []byte) error {
	type t MockItem
	return json.Unmarshal(b, (*t)(m))
}

type MockQueue struct {
	Q []*MockItem
}

type MockCall MockItem

func (mc *MockCall) MarshalJSON() ([]byte, error) {
	item := (*MockItem)(mc)
	q := MockQueue{
		Q: []*MockItem{item},
	}
	return json.Marshal(q)
}

func (mc *MockCall) NextMockItem() *MockItem {
	return (*MockItem)(mc)
}

func (mc *MockQueue) NextMockItem() *MockItem {
	if len(mc.Q) == 0 {
		return &MockItem{
			MockError: "no more calls expected",
		}
	}

	var mi *MockItem
	mi, mc.Q = mc.Q[0], mc.Q[1:]

	return mi
}

func MockExecutor(md MockItemDispenser) ExecutorFactory {
	return func(_ string) *Executor {
		if md == nil {
			md = &MockCall{
				MockError: "no mocks provided",
			}
		}

		path, ctxFunc := mockCommandCtxFunc(md)
		executor := NewExecutor(path)
		executor.cmdCtxFunc = ctxFunc
		return executor
	}
}

func mockCommandCtxFunc(md MockItemDispenser) (string, cmdCtxFunc) {
	return os.Args[0], func(ctx context.Context, path string, arg ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], os.Args[1:]...)

		b, err := md.NextMockItem().MarshalJSON()
		if err != nil {
			panic(err)
		}
		expectedJson := string(b)
		cmd.Env = []string{"TF_LS_MOCK=" + string(expectedJson)}

		return cmd
	}
}

func ExecuteMockData(rawMockData string) int {
	mi := &MockItem{}
	err := mi.UnmarshalJSON([]byte(rawMockData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to unmarshal mock response: %s", err)
		return 1
	}
	return validateMockItem(mi, os.Args[1:], os.Stdout, os.Stderr)
}

func validateMockItem(m *MockItem, args []string, stdout, stderr io.Writer) int {
	if m.MockError != "" {
		fmt.Fprintf(stderr, m.MockError)
		return 1
	}

	givenArgs := args
	if !reflect.DeepEqual(m.Args, givenArgs) {
		fmt.Fprintf(stderr,
			"arguments don't match.\nexpected: %q\ngiven: %q\n",
			m.Args, givenArgs)
		return 1
	}

	if m.SleepDuration > 0 {
		time.Sleep(m.SleepDuration)
	}

	fmt.Fprint(stdout, m.Stdout)
	fmt.Fprint(stderr, m.Stderr)

	return m.ExitCode
}
