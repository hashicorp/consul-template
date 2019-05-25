// +build !go1.10

package cmdflag

import (
	"flag"
	"io"
	"os"
)

func fsetOutput(fs *flag.FlagSet) io.Writer {
	return os.Stderr
}
