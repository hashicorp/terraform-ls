package decoder

import (
	"context"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/reference"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/features/stacks/state"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	stackschema "github.com/hashicorp/terraform-schema/schema"
)

type PathReader struct {
	StateReader StateReader
}

type StateReader interface {
	List() ([]*state.StackRecord, error)
	StackRecordByPath(modPath string) (*state.StackRecord, error)
}

// PathContext returns a Stacks PathContext for the given path
func (pr *PathReader) PathContext(path lang.Path) (*decoder.PathContext, error) {
	record, err := pr.StateReader.StackRecordByPath(path.Path)
	if err != nil {
		return nil, err
	}

	// get terrafom version from statereader and use that to get the schema

	// TODO: This only provides tfstacks schema. There is also tfdeploy schema
	// TODO: this should only work for terraform 1.8 and above
	var schema *schema.BodySchema
	switch path.LanguageID {
	case lsp.Stacks.String():
		schema = stackschema.CoreStackSchema(stackschema.LatestAvailableVersion)
	case lsp.Deploy.String():
		schema = stackschema.CoreDeploySchema(stackschema.LatestAvailableVersion)
	}

	pathCtx := &decoder.PathContext{
		Schema:           schema,
		ReferenceOrigins: make(reference.Origins, 0),
		ReferenceTargets: make(reference.Targets, 0),
		Files:            make(map[string]*hcl.File, 0),
	}

	// TODO: Add reference origins and targets
	// for _, origin := range record.RefOrigins {
	// 	if ast.IsStacksFilename(origin.OriginRange().Filename) {
	// 		pathCtx.ReferenceOrigins = append(pathCtx.ReferenceOrigins, origin)
	// 	}
	// }
	// for _, target := range record.RefTargets {
	// 	if target.RangePtr != nil && ast.IsStacksFilename(target.RangePtr.Filename) {
	// 		pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
	// 	} else if target.RangePtr == nil {
	// 		pathCtx.ReferenceTargets = append(pathCtx.ReferenceTargets, target)
	// 	}
	// }

	for name, f := range record.ParsedStackFiles {
		pathCtx.Files[name.String()] = f
	}
	
	for name, f := range record.ParsedDeployFiles {
		pathCtx.Files[name.String()] = f
	}

	return pathCtx, nil
}

func (pr *PathReader) Paths(ctx context.Context) []lang.Path {
	paths := make([]lang.Path, 0)

	stackRecords, err := pr.StateReader.List()
	if err != nil {
		return paths
	}

	for _, record := range stackRecords {
		paths = append(paths, lang.Path{
			Path:       record.Path(),
			LanguageID: lsp.Stacks.String(),
		})
	}

	return paths
}
