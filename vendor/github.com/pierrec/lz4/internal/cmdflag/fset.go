// +build go1.10

package cmdflag

import (
	"flag"
	"io"
)

func fsetOutput(fs *flag.FlagSet) io.Writer {
	return fs.Output()
}
