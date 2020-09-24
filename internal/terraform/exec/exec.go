package exec

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

var defaultExecTimeout = 30 * time.Second

// ExecutorFactory can be used in external consumers of exec pkg
// to enable easy swapping with MockExecutor
type ExecutorFactory func(path string) *Executor

type Executor struct {
	tf *tfexec.Terraform

	timeout time.Duration

	execPath    string
	workDir     string
	logger      *log.Logger
	execLogPath string
}

type command struct {
	Cmd          *exec.Cmd
	Context      context.Context
	CancelFunc   context.CancelFunc
	StdoutBuffer *bytes.Buffer
	StderrBuffer *bytes.Buffer
}

func NewExecutor(path string) *Executor {
	return &Executor{
		timeout:  defaultExecTimeout,
		execPath: path,
		logger:   log.New(ioutil.Discard, "", 0),
	}
}

// tfExec should be used to initialize and return an instance of tfexec.Terraform just in
// time, the instance will be cached. This is helpful because the language server sets the working
// directory after the creation of the Executor instance.
func (e *Executor) tfExec() *tfexec.Terraform {
	if e.tf == nil {
		tf, err := tfexec.NewTerraform(e.workDir, e.execPath)
		if err != nil {
			panic(err)
		}
		tf.SetLogger(e.logger)

		// TODO: support log filename template upstream
		// if e.execLogPath != "" {
		// 	logPath, err := logging.ParseExecLogPath(cmd.Args, e.execLogPath)
		// 	tf.SetLogPath(logPath)
		// }

		e.tf = tf
	}
	return e.tf
}

func (e *Executor) SetLogger(logger *log.Logger) {
	e.logger = logger
}

func (e *Executor) SetExecLogPath(path string) {
	e.execLogPath = path
}

func (e *Executor) SetTimeout(duration time.Duration) {
	e.timeout = duration
}

func (e *Executor) SetWorkdir(workdir string) {
	e.workDir = workdir
}

func (e *Executor) GetExecPath() string {
	return e.execPath
}

type Formatter func(ctx context.Context, input []byte) ([]byte, error)

func (e *Executor) FormatterForVersion(v string) (Formatter, error) {
	return formatterForVersion(v, e.Format)
}

func formatterForVersion(v string, f Formatter) (Formatter, error) {
	if v == "" {
		return nil, fmt.Errorf("unknown version - unable to provide formatter")
	}

	ver, err := version.NewVersion(v)
	if err != nil {
		return nil, err
	}

	// "fmt" command was first introduced in v0.7.7
	fmtCapableVersion, err := version.NewVersion("0.7.7")
	if err != nil {
		return nil, err
	}

	if ver.GreaterThanOrEqual(fmtCapableVersion) {
		return f, nil
	}

	return nil, fmt.Errorf("no formatter available for %s", v)
}

func (e *Executor) Format(ctx context.Context, input []byte) ([]byte, error) {
	ctx, cancel := contextWithTimeout(ctx, e.timeout)
	defer cancel()

	formatted, err := tfexec.FormatString(ctx, e.execPath, string(input))
	return []byte(formatted), err
}

func (e *Executor) Version(ctx context.Context) (string, error) {
	ctx, cancel := contextWithTimeout(ctx, e.timeout)
	defer cancel()

	v, _, err := e.tfExec().Version(ctx, true)
	if err != nil {
		return "", err
	}
	// TODO: consider refactoring codebase to work directly with go-version.Version
	return v.String(), nil
}

func (e *Executor) VersionIsSupported(ctx context.Context, c version.Constraints) error {
	v, err := e.Version(ctx)
	if err != nil {
		return err
	}
	ver, err := version.NewVersion(v)
	if err != nil {
		return err
	}

	if !c.Check(ver) {
		return fmt.Errorf("version %s not supported (%s)",
			ver.String(), c.String())
	}

	return nil
}

func (e *Executor) ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error) {
	ctx, cancel := contextWithTimeout(ctx, e.timeout)
	defer cancel()

	return e.tfExec().ProvidersSchema(ctx)
}

func contextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	return ctx, cancel
}
