package diagnostics

import (
	"testing"

	"golang.org/x/net/context"
)

func TestDiagnoseHCL_Closes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewNotifier(ctx)
	cancel()
	n.DiagnoseHCL(context.Background(), "", []byte{})
	if _, open := <-n.hclDocs; open {
		t.Fatal("documents channel should be closed")
	}
}

func TestDiagnoseHCL_DoesNotSendAfterClose(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatal(err)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	n := NewNotifier(ctx)
	cancel()
	n.DiagnoseHCL(context.Background(), "", []byte{})
	n.DiagnoseHCL(context.Background(), "", []byte{})
}

func TestHCLParse_ReturnsEmptySliceWhenValid(t *testing.T) {
	diags := hclParse(documentContext{ctx: context.Background(), uri: "test", text: hcl(`provider "test" {}`)})
	if diags == nil {
		t.Fatal("slice needs to be initialized")
	}
	if len(diags) > 0 {
		t.Fatalf("valid hcl should return an empty slice: %v", diags)
	}
}

func TestHCLParse_ReturnsDiagsWhenInvalid(t *testing.T) {
	diags := hclParse(documentContext{ctx: context.Background(), uri: "test", text: hcl(`provider test" {}`)})
	if len(diags) == 0 {
		t.Fatal("invalid hcl should return diags")
	}
}

func hcl(text string) []byte {
	return append([]byte(text), '\n')
}
