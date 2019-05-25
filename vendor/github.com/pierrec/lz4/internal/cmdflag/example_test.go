package cmdflag_test

import (
	"flag"
	"fmt"
	"os"

	"github.com/pierrec/lz4/internal/cmdflag"
)

func ExampleParse() {
	// Declare the `split` cmdflag.
	cmdflag.New(
		"split",
		"argsdesc",
		"desc",
		flag.ExitOnError,
		func(fs *flag.FlagSet) cmdflag.Handler {
			// Declare the cmdflag specific flags.
			var s string
			fs.StringVar(&s, "s", "", "string to be split")

			// Return the handler to be executed when the cmdflag is found.
			return func(...string) error {
				i := len(s) / 2
				fmt.Println(s[:i], s[i:])
				return nil
			}
		})

	// The following is only used to emulate passing command line arguments to `program`.
	// It is equivalent to running:
	// ./program split -s hello
	args := os.Args
	defer func() { os.Args = args }()
	os.Args = []string{"program", "split", "-s", "hello"}

	// Process the command line arguments.
	if err := cmdflag.Parse(); err != nil {
		panic(err)
	}

	// Output:
	// he llo
}
