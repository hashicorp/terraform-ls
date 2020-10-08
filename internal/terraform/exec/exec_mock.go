package exec

import (
	exec_mock "github.com/hashicorp/terraform-ls/internal/terraform/exec/mock"
	"github.com/stretchr/testify/mock"
)

func NewMockExecutor(calls []*mock.Call) ExecutorFactory {
	return func(string, string) (TerraformExecutor, error) {
		me := &exec_mock.Executor{}
		firstCalls := []*mock.Call{
			{
				Method:        "SetLogger",
				Arguments:     []interface{}{mock.Anything},
				Repeatability: 1,
			},
		}
		me.ExpectedCalls = append(firstCalls, calls...)
		return me, nil
	}
}
