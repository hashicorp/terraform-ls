// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package module

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/terraform/exec"
)

func TerraformExecutorForModule(ctx context.Context, modPath string) (exec.TerraformExecutor, error) {
	newExecutor, ok := exec.ExecutorFactoryFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no terraform executor provided")
	}

	execPath, err := TerraformExecPath(ctx)
	if err != nil {
		return nil, err
	}

	tfExec, err := newExecutor(modPath, execPath)
	if err != nil {
		return nil, err
	}

	opts, ok := exec.ExecutorOptsFromContext(ctx)
	if ok && opts.ExecLogPath != "" {
		tfExec.SetExecLogPath(opts.ExecLogPath)
	}
	if ok && opts.Timeout != 0 {
		tfExec.SetTimeout(opts.Timeout)
	}

	return tfExec, nil
}

func TerraformExecPath(ctx context.Context) (string, error) {
	opts, ok := exec.ExecutorOptsFromContext(ctx)
	if ok && opts.ExecPath != "" {
		return opts.ExecPath, nil
	} else {
		return "", NoTerraformExecPathErr{}
	}
}
