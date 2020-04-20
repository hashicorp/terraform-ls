package lang

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestDatasourceBlock_Name(t *testing.T) {
	testCases := []struct {
		name string
		src  string

		expectedName string
		expectedErr  error
	}{
		{
			"invalid config - no label",
			`data {
}
`,
			"<unknown>",
			nil,
		},
		{
			"invalid config - single label",
			`data "aws_instance" {
}
`,
			"aws_instance.<unknown>",
			nil,
		},
		{
			"invalid config - three labels",
			`data "aws_instance" "name" "extra" {
}
`,
			"aws_instance.name",
			nil,
		},
		{
			"valid config",
			`data "aws_instance" "name" {
}
`,
			"aws_instance.name",
			nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.name), func(t *testing.T) {
			block, err := AsHCLSyntaxBlock(parseHclBlock(t, tc.src))
			if err != nil {
				t.Fatal(err)
			}

			pf := &datasourceBlockFactory{logger: log.New(os.Stdout, "", 0)}
			p, err := pf.New(block)

			if err != nil {
				if tc.expectedErr != nil && err.Error() == tc.expectedErr.Error() {
					return
				}
				t.Fatalf("Errors don't match.\nexpected: %#v\ngiven: %#v",
					tc.expectedErr, err)
			}
			if tc.expectedErr != nil {
				t.Fatalf("Expected error: %#v", tc.expectedErr)
			}

			name := p.Name()
			if name != tc.expectedName {
				t.Fatalf("Name doesn't match.\nexpected: %q\ngiven: %q",
					tc.expectedName, name)
			}
		})
	}
}
