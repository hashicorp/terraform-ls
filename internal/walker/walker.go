// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package walker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/hashicorp/terraform-ls/internal/document"
	"github.com/hashicorp/terraform-ls/internal/eventbus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	discardLogger = log.New(io.Discard, "", 0)

	// skipDirNames represent directory names which would never contain
	// plugin/module cache, so it's safe to skip them during the walk
	//
	// please keep the list in `SETTINGS.md` in sync
	skipDirNames = map[string]bool{
		".git":                true,
		".idea":               true,
		".vscode":             true,
		"terraform.tfstate.d": true,
		".terragrunt-cache":   true,
	}
)

type Walker struct {
	fs        fs.ReadDirFS
	pathStore PathStore

	logger   *log.Logger
	eventBus *eventbus.EventBus

	Collector *WalkerCollector

	cancelFunc context.CancelFunc

	ignoredPaths          map[string]bool
	ignoredDirectoryNames map[string]bool
}

type PathStore interface {
	AwaitNextDir(ctx context.Context) (context.Context, document.DirHandle, error)
	RemoveDir(dir document.DirHandle) error
}

const tracerName = "github.com/hashicorp/terraform-ls/internal/walker"

func NewWalker(fs fs.ReadDirFS, pathStore PathStore, eventBus *eventbus.EventBus) *Walker {
	return &Walker{
		fs:                    fs,
		pathStore:             pathStore,
		logger:                discardLogger,
		ignoredDirectoryNames: skipDirNames,
		eventBus:              eventBus,
	}
}

func (w *Walker) SetLogger(logger *log.Logger) {
	w.logger = logger
}

func (w *Walker) SetIgnoredPaths(ignoredPaths []string) {
	w.ignoredPaths = make(map[string]bool)
	for _, path := range ignoredPaths {
		w.ignoredPaths[path] = true
	}
}

func (w *Walker) SetIgnoredDirectoryNames(ignoredDirectoryNames []string) {
	if w.cancelFunc != nil {
		panic("cannot set ignorelist after walking started")
	}
	for _, path := range ignoredDirectoryNames {
		w.ignoredDirectoryNames[path] = true
	}
}

func (w *Walker) Stop() {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
}

func (w *Walker) StartWalking(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	w.cancelFunc = cancelFunc

	go func() {
		for {
			pathCtx, nextDir, err := w.pathStore.AwaitNextDir(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				w.logger.Printf("walker: awaiting next dir failed: %s", err)
				w.collectError(err)
				return
			}

			spanCtx := trace.SpanContextFromContext(pathCtx)

			ctx = trace.ContextWithSpanContext(ctx, spanCtx)

			tracer := otel.Tracer(tracerName)
			ctx, span := tracer.Start(ctx, "walk-path", trace.WithAttributes(attribute.KeyValue{
				Key:   attribute.Key("URI"),
				Value: attribute.StringValue(nextDir.URI),
			}))

			err = w.walk(ctx, nextDir)
			if err != nil {
				w.logger.Printf("walker: walking through %q failed: %s", nextDir, err)
				w.collectError(err)
				span.RecordError(err)
				span.SetStatus(codes.Error, "walking failed")
				span.End()
				continue
			}
			span.SetStatus(codes.Ok, "walking finished")
			span.End()

			err = w.pathStore.RemoveDir(nextDir)
			if err != nil {
				w.logger.Printf("walker: removing dir %q from queue failed: %s", nextDir, err)
				w.collectError(err)
				span.RecordError(err)
				span.SetStatus(codes.Error, "walking failed")
				span.End()
				continue
			}

			w.logger.Printf("walker: walking through %q finished", nextDir)

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return nil
}

func (w *Walker) collectError(err error) {
	if w.Collector != nil {
		w.Collector.CollectError(err)
	}
}

func (w *Walker) isSkippableDir(dirName string) bool {
	_, ok := w.ignoredDirectoryNames[dirName]
	return ok
}

func (w *Walker) walk(ctx context.Context, dir document.DirHandle) error {
	if _, ok := w.ignoredPaths[dir.Path()]; ok {
		w.logger.Printf("skipping walk due to dir being excluded: %s", dir.Path())
		return nil
	}

	dirEntries, err := fs.ReadDir(w.fs, dir.Path())
	if err != nil {
		w.logger.Printf("reading directory failed: %s: %s", dir.Path(), err)
		// fs.ReadDir (or at least the os.ReadDir implementation) returns
		// the entries it was able to read before the error, along with the error.
	}

	files := make([]string, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		files = append(files, dirEntry.Name())
	}

	w.eventBus.Discover(eventbus.DiscoverEvent{
		Path:  dir.Path(),
		Files: files,
	})

	for _, dirEntry := range dirEntries {
		select {
		case <-ctx.Done():
			w.logger.Printf("cancelling walk of %s...", dir)
			return fmt.Errorf("walk cancelled")
		default:
		}

		if w.isSkippableDir(dirEntry.Name()) {
			w.logger.Printf("skipping ignored dir name: %s", dirEntry.Name())
			continue
		}

		if dirEntry.IsDir() {
			path := filepath.Join(dir.Path(), dirEntry.Name())
			dirHandle := document.DirHandleFromPath(path)
			err = w.walk(ctx, dirHandle)
			if err != nil {
				return err
			}
		}
	}
	w.logger.Printf("walking of %s finished", dir)
	return err
}
