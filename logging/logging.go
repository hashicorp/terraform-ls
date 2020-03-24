package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

func NewLogger(w io.Writer) *log.Logger {
	return log.New(w, "", log.LstdFlags|log.Lshortfile)
}

type fileLogger struct {
	l *log.Logger
	f *os.File
}

func NewFileLogger(rawPath string) (*fileLogger, error) {
	path, err := parseLogPath(rawPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("please provide absolute log path to prevent ambiguity (given: %q)",
			path)
	}

	mode := os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	file, err := os.OpenFile(path, mode, 0600)
	if err != nil {
		return nil, err
	}

	return &fileLogger{
		l: NewLogger(file),
		f: file,
	}, nil
}

func parseLogPath(rawPath string) (string, error) {
	tpl := template.New("log-file")
	tpl = tpl.Funcs(template.FuncMap{
		"timestamp": time.Now().Local().Unix,
		"pid":       os.Getpid,
		"ppid":      os.Getppid,
	})
	tpl, err := tpl.Parse(rawPath)
	if err != nil {
		return "", err
	}

	buf := &strings.Builder{}
	err = tpl.Execute(buf, nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ValidateExecLogPath(rawPath string) error {
	_, err := parseExecLogPathTemplate([]string{}, rawPath)
	return err
}

func ParseExecLogPath(args []string, rawPath string) (string, error) {
	tpl, err := parseExecLogPathTemplate(args, rawPath)
	if err != nil {
		return "", err
	}

	buf := &strings.Builder{}
	err = tpl.Execute(buf, nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func parseExecLogPathTemplate(args []string, rawPath string) (*template.Template, error) {
	tpl := template.New("tf-log-file")
	tpl = tpl.Funcs(template.FuncMap{
		"timestamp": time.Now().Local().Unix,
		"lsPid":     os.Getpid,
		"lsPpid":    os.Getppid,
		"args": func() string {
			return escapeArguments(args)
		},
	})
	return tpl.Parse(rawPath)
}

// escapeArguments turns arguments into a string
// which is safe to use in a filename without any side-effects
func escapeArguments(rawArgs []string) string {
	unsafeCharsRe := regexp.MustCompile(`[^a-z-_]+`)

	safeArgs := make([]string, len(rawArgs), len(rawArgs))
	for _, rawArg := range rawArgs {
		// Replace any unsafe character with a hyphen
		safeArg := unsafeCharsRe.ReplaceAllString(rawArg, "-")
		safeArgs = append(safeArgs, safeArg)
	}

	args := strings.Join(safeArgs, "-")

	// Reduce hyphens to just one
	hyphensRe := regexp.MustCompile(`[-]+`)
	reduced := hyphensRe.ReplaceAllString(args, "-")

	// Trim hyphens from both ends
	return strings.Trim(reduced, "-")
}

func (fl *fileLogger) Logger() *log.Logger {
	return fl.l
}

func (fl *fileLogger) Close() error {
	return fl.f.Close()
}
