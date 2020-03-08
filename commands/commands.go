package commands

import (
	"flag"
	"io/ioutil"
)

func defaultFlagSet(cmdName string) *flag.FlagSet {
	f := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {}

	return f
}
