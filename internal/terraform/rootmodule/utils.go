package rootmodule

import (
	"path/filepath"
	"strings"
)

func pathEquals(path1, path2 string) bool {
	volumn1 := filepath.VolumeName(path1)
	volumn2 := filepath.VolumeName(path2)
	return strings.EqualFold(volumn1, volumn2) && path1[len(volumn1)] == path2[len(volumn2)]
}
