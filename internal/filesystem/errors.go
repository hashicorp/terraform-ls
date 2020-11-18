package filesystem

import (
	"fmt"
)

type DocumentNotOpenErr struct {
	DocumentHandler DocumentHandler
}

func (e *DocumentNotOpenErr) Error() string {
	return fmt.Sprintf("document is not open: %s", e.DocumentHandler.URI())
}

type MetadataAlreadyExistsErr struct {
	DocumentHandler DocumentHandler
}

func (e *MetadataAlreadyExistsErr) Error() string {
	return fmt.Sprintf("document metadata already exists: %s", e.DocumentHandler.URI())
}

type UnknownDocumentErr struct {
	DocumentHandler DocumentHandler
}

func (e *UnknownDocumentErr) Error() string {
	return fmt.Sprintf("unknown document: %s", e.DocumentHandler.URI())
}

type InvalidPosErr struct {
	Pos Pos
}

func (e *InvalidPosErr) Error() string {
	return fmt.Sprintf("invalid position: %s", e.Pos)
}
