// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pathcmp

import (
	"path/filepath"
	"strings"
)

func PathEquals(path1, path2 string) bool {
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)
	volume1 := filepath.VolumeName(path1)
	volume2 := filepath.VolumeName(path2)
	return strings.EqualFold(volume1, volume2) && path1[len(volume1):] == path2[len(volume2):]
}
