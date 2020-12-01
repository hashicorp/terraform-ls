package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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
	_, err := parseExecLogPathTemplate("", rawPath)
	return err
}

func ParseExecLogPath(method string, rawPath string) (string, error) {
	tpl, err := parseExecLogPathTemplate(method, rawPath)
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

func parseExecLogPathTemplate(method string, rawPath string) (*template.Template, error) {
	tpl := template.New("tf-log-file")
	methodFunc := func() string {
		return method
	}
	tpl = tpl.Funcs(template.FuncMap{
		"timestamp": time.Now().Local().Unix,
		"lsPid":     os.Getpid,
		"lsPpid":    os.Getppid,
		"method":    methodFunc,

		// DEPRECATED
		"args": methodFunc,
	})
	return tpl.Parse(rawPath)
}

func (fl *fileLogger) Logger() *log.Logger {
	return fl.l
}

func (fl *fileLogger) Close() error {
	return fl.f.Close()
}
