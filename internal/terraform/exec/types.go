package exec

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// ExecutorFactory can be used in external consumers of exec pkg
// to enable easy swapping with MockExecutor
type ExecutorFactory func(workDir, execPath string) (TerraformExecutor, error)

type Formatter func(ctx context.Context, input []byte) ([]byte, error)

//go:generate mockery --name TerraformExecutor --structname Executor --filename executor.go --outpkg mock --output ./mock

type TerraformExecutor interface {
	SetLogger(logger *log.Logger)
	SetExecLogPath(path string) error
	SetTimeout(duration time.Duration)
	GetExecPath() string
	Init(ctx context.Context, opts ...tfexec.InitOption) error
	Format(ctx context.Context, input []byte) ([]byte, error)
	Plan(ctx context.Context) error
	Version(ctx context.Context) (*version.Version, map[string]*version.Version, error)
	Validate(ctx context.Context) ([]tfjson.Diagnostic, error)
	ProviderSchemas(ctx context.Context) (*tfjson.ProviderSchemas, error)
}
