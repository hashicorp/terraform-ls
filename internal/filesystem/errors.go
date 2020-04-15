package filesystem

import (
	"fmt"
)

type FileNotOpenErr struct {
	FileHandler FileHandler
}

func (e *FileNotOpenErr) Error() string {
	return fmt.Sprintf("file is not open: %s", e.FileHandler.DocumentURI())
}
