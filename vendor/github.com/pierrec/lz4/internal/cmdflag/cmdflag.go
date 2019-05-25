// Package cmdflag adds single level cmdflag support to the standard library flag package.
package cmdflag

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

//TODO add a -fullversion command to display module versions, compiler version
//TODO add multi level command support

// VersionBoolFlag is the flag name to be used as a boolean flag to display the program version.
const VersionBoolFlag = "version"

// Usage is the function used for help.
var Usage = func() {
	fset := flag.CommandLine
	out := fsetOutput(fset)

	program := programName(os.Args[0])
	fmt.Fprintf(out, "Usage of %s:\n", program)
	fset.PrintDefaults()

	fmt.Fprintf(out, "\nSubcommands:")
	for _, sub := range subs {
		fmt.Fprintf(out, "\n%s\n%s %s\n", sub.desc, sub.name, sub.argsdesc)
		fs, _ := sub.init(out)
		fs.PrintDefaults()
	}
}

// Handler is the function called when a matching cmdflag is found.
type Handler func(...string) error

type subcmd struct {
	errh     flag.ErrorHandling
	name     string
	argsdesc string
	desc     string
	ini      func(*flag.FlagSet) Handler
}

func (sub *subcmd) init(out io.Writer) (*flag.FlagSet, Handler) {
	fname := fmt.Sprintf("cmdflag `%s`", sub.name)
	fs := flag.NewFlagSet(fname, sub.errh)
	fs.SetOutput(out)
	return fs, sub.ini(fs)
}

var (
	mu   sync.Mutex
	subs []*subcmd
)

// New instantiates a new cmdflag with its name and description.
//
// It is safe to be called from multiple go routines (typically in init functions).
//
// The cmdflag initializer is called only when the cmdflag is present on the command line.
// The handler is called with the remaining arguments once the cmdflag flags have been parsed successfully.
//
// It will panic if the cmdflag already exists.
func New(name, argsdesc, desc string, errh flag.ErrorHandling, initializer func(*flag.FlagSet) Handler) {
	sub := &subcmd{
		errh:     errh,
		name:     name,
		argsdesc: argsdesc,
		desc:     desc,
		ini:      initializer,
	}

	mu.Lock()
	defer mu.Unlock()
	for _, sub := range subs {
		if sub.name == name {
			panic(fmt.Errorf("cmdflag %s redeclared", name))
		}
	}
	subs = append(subs, sub)
}

// Parse parses the command line arguments including the global flags and, if any, the cmdflag and its flags.
//
// If the VersionBoolFlag is defined as a global boolean flag, then the program version is displayed and the program stops.
func Parse() error {
	args := os.Args
	if len(args) == 1 {
		return nil
	}

	// Global flags.
	fset := flag.CommandLine
	fset.Usage = Usage
	out := fsetOutput(fset)

	if err := fset.Parse(args[1:]); err != nil {
		return err
	}

	// Handle version request.
	if f := fset.Lookup(VersionBoolFlag); f != nil {
		if v, ok := f.Value.(flag.Getter); ok {
			// All values implemented by the flag package implement the flag.Getter interface.
			if b, ok := v.Get().(bool); ok && b {
				// The flag was defined as a bool and is set.
				program := programName(args[0])
				fmt.Fprintf(out, "%s version %s %s/%s\n",
					program, buildinfo(),
					runtime.GOOS, runtime.GOARCH)
				return nil
			}
		}
	}

	// No cmdflag.
	if fset.NArg() == 0 {
		return nil
	}

	// Subcommand.
	idx := len(args) - fset.NArg()
	s := args[idx]
	args = args[idx+1:]
	for _, sub := range subs {
		if sub.name != s {
			continue
		}

		fs, handler := sub.init(out)
		if err := fs.Parse(args); err != nil {
			return err
		}
		return handler(args[len(args)-fs.NArg():]...)
	}

	return fmt.Errorf("%s is not a valid cmdflag", s)
}

func programName(s string) string {
	name := filepath.Base(s)
	return strings.TrimSuffix(name, ".exe")
}
