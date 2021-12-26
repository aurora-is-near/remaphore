package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/aurora-is-near/remaphore/src/config"

	"github.com/aurora-is-near/remaphore/src/protocol"
)

func CleanStrings(s ...string) []string {
	r := make([]string, 0, len(s))
	for _, v := range s {
		t := strings.TrimFunc(v, unicode.IsSpace)
		if len(t) == 0 {
			continue
		}
		r = append(r, t)
	}
	return r
}

func StdErr(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, v...)
}

func StdOut(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(os.Stdout, format, v...)
}

func baseName() string {
	return path.Base(os.Args[0])
}

func combineSlices(a, b []interface{}) []interface{} {
	return append(a, b...)
}

func ExitError(exitCode int, format string, v ...interface{}) {
	StdErr("%s: "+format+"\n", combineSlices([]interface{}{baseName()}, v)...)
	os.Exit(exitCode)
}

// func ExitOut(exitCode int, format string, v ...interface{}) {
// 	StdOut("%s: "+format+"\n", combineSlices([]interface{}{baseName()}, v)...)
// 	os.Exit(exitCode)
// }

func PrintConfig() {
	c := protocol.NewConfig()
	StdOut("%s\n", c)
	os.Exit(0)
}

func GetConfig(filename string) *protocol.Config {
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		ExitError(2, "ERROR: %s", err)
	}
	ret, err := config.ParseConfig(d)
	if err != nil {
		ExitError(2, "ERROR: %s", err)
	}
	return ret
}
