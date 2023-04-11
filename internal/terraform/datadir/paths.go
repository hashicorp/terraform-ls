// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datadir

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const DataDirName = ".terraform"

func watchableModuleDirs(modPath string) []string {
	return []string{
		filepath.Join(modPath, DataDirName),
		filepath.Join(modPath, DataDirName, "modules"),
		filepath.Join(modPath, DataDirName, "plugins"),
		filepath.Join(modPath, DataDirName, "plugins", runtime.GOOS+"_"+runtime.GOARCH),
	}
}

type EventType rune

const (
	AnyEventType    EventType = '*'
	CreateEventType EventType = 'c'
	ModifyEventType EventType = 'm'
	DeleteEventType EventType = 'd'
)

type WatchPattern struct {
	Pattern   string
	EventType EventType
}

func PathGlobPatternsForWatching() []WatchPattern {
	patterns := make([]WatchPattern, 0)

	// This is necessary because clients may not send delete notifications
	// for individual nested files when the parent directory is deleted.
	// VS Code / vscode-languageclient behaves this way.
	patterns = append(patterns, WatchPattern{
		Pattern:   "**/" + DataDirName,
		EventType: DeleteEventType,
	})

	patterns = append(patterns, WatchPattern{
		Pattern:   "**/" + path.Join(manifestPathElements...),
		EventType: AnyEventType,
	})
	for _, pElems := range pluginLockFilePathElements {
		patterns = append(patterns, WatchPattern{
			Pattern:   "**/" + path.Join(pElems...),
			EventType: AnyEventType,
		})
	}

	return patterns
}

func ModuleUriFromDataDir(rawUri string) (string, bool) {
	suffix := "/" + DataDirName
	if strings.HasSuffix(rawUri, suffix) {
		return strings.TrimSuffix(rawUri, suffix), true
	}
	return "", false
}

func ModuleUriFromPluginLockFile(rawUri string) (string, bool) {
	for _, pathElems := range pluginLockFilePathElements {
		suffix := "/" + path.Join(pathElems...)
		if strings.HasSuffix(rawUri, suffix) {
			return strings.TrimSuffix(rawUri, suffix), true
		}
	}
	return "", false
}

func ModuleUriFromModuleLockFile(rawUri string) (string, bool) {
	suffix := "/" + path.Join(manifestPathElements...)
	if strings.HasSuffix(rawUri, suffix) {
		return strings.TrimSuffix(rawUri, suffix), true
	}
	return "", false
}
