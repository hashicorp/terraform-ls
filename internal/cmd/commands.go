// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"flag"
	"io/ioutil"
	"strings"
)

func defaultFlagSet(cmdName string) *flag.FlagSet {
	f := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {}

	return f
}

func helpForFlags(fs *flag.FlagSet) string {
	buf := &strings.Builder{}
	buf.WriteString("Options:\n\n")

	w := fs.Output()
	defer fs.SetOutput(w)
	fs.SetOutput(buf)
	fs.PrintDefaults()

	return buf.String()
}
