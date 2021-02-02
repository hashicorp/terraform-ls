package module

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TerraformExecutorForModule(ctx context.Context, mod Module) (exec.TerraformExecutor, error) {
	newExecutor, ok := exec.ExecutorFactoryFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no terraform executor provided")
	}

	var tfExec exec.TerraformExecutor
	var err error

	opts, ok := exec.ExecutorOptsFromContext(ctx)
	if ok && opts.ExecPath != "" {
		tfExec, err = newExecutor(mod.Path(), opts.ExecPath)
		if err != nil {
			return nil, err
		}
	} else if mod.TerraformExecPath() != "" {
		tfExec, err = newExecutor(mod.Path(), mod.TerraformExecPath())
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no exec path provided for terraform")
	}

	if ok && opts.ExecLogPath != "" {
		tfExec.SetExecLogPath(opts.ExecLogPath)
	}
	if ok && opts.Timeout != 0 {
		tfExec.SetTimeout(opts.Timeout)
	}

	return tfExec, nil
}
