package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
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

type mockExecutor struct {
	path    string
	timeout time.Duration
	md      MockItemDispenser
}

func (*mockExecutor) SetWorkdir(string) {}

func (*mockExecutor) SetExecLogPath(string) {}

func (*mockExecutor) SetLogger(*log.Logger) {}

func (m *mockExecutor) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

func (m *mockExecutor) GetExecPath() string {
	return m.path
}

func (m *mockExecutor) Version(ctx context.Context) (string, error) {
	out, err := m.run(ctx, "version")
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}
	outString := string(out)
	lines := strings.Split(outString, "\n")
	if len(lines) < 1 {
		return "", fmt.Errorf("unexpected version output: %q", outString)
	}
	version := strings.TrimPrefix(lines[0], "Terraform v")

	return version, nil
}

func (m *mockExecutor) Format(ctx context.Context, input []byte) ([]byte, error) {
	return m.run(ctx, "fmt", "-")
}

func (m *mockExecutor) ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error) {
	outBytes, err := m.run(ctx, "providers", "schema", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	var schemas tfjson.ProviderSchemas
	err = json.Unmarshal(outBytes, &schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &schemas, nil
}

func (m *mockExecutor) FormatterForVersion(v string) (Formatter, error) {
	return formatterForVersion(v, m.Format)
}

func (m *mockExecutor) run(ctx context.Context, args ...string) ([]byte, error) {
	b, err := m.md.NextMockItem().MarshalJSON()
	if err != nil {
		panic(err)
	}

	cancel := func() {}
	if m.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
	}
	defer cancel()

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	cmd := exec.CommandContext(ctx, os.Args[0])
	cmd.Args = append([]string{"terraform"}, args...)
	cmd.Env = []string{"TF_LS_MOCK=" + string(b)}
	cmd.Stderr = &errBuf
	cmd.Stdout = &outBuf
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		if tErr, ok := err.(*exec.ExitError); ok {
			exitErr := &ExitError{
				Err:    tErr,
				Path:   cmd.Path,
				Stdout: outBuf.String(),
				Stderr: errBuf.String(),
			}

			ctxErr := ctx.Err()
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				exitErr.CtxErr = ExecTimeoutError(cmd.Args, m.timeout)
			}
			if errors.Is(ctxErr, context.Canceled) {
				exitErr.CtxErr = ExecCanceledError(cmd.Args)
			}

			return nil, exitErr
		}

		return nil, err
	}

	return outBuf.Bytes(), nil
}

type MockExecutorFactory func(string) *mockExecutor

func MockExecutor(md MockItemDispenser) MockExecutorFactory {
	return func(string) *mockExecutor {
		if md == nil {
			md = &MockCall{
				MockError: "no mocks provided",
			}
		}

		return &mockExecutor{path: os.Args[0], md: md}
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
