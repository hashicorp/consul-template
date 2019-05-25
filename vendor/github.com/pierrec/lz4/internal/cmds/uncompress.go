package cmds

import (
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v2"
	"io"
	"os"
	"strings"

	"github.com/pierrec/lz4"
	"github.com/pierrec/lz4/internal/cmdflag"
)

// Uncompress uncompresses a set of files or from stdin to stdout.
func Uncompress(_ *flag.FlagSet) cmdflag.Handler {
	return func(args ...string) error {
		zr := lz4.NewReader(nil)

		// Use stdin/stdout if no file provided.
		if len(args) == 0 {
			zr.Reset(os.Stdin)
			_, err := io.Copy(os.Stdout, zr)
			return err
		}

		for _, zfilename := range args {
			// Input file.
			zfile, err := os.Open(zfilename)
			if err != nil {
				return err
			}
			zinfo, err := zfile.Stat()
			if err != nil {
				return err
			}
			mode := zinfo.Mode() // use the same mode for the output file

			// Output file.
			filename := strings.TrimSuffix(zfilename, lz4.Extension)
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			zr.Reset(zfile)

			zfinfo, err := zfile.Stat()
			if err != nil {
				return err
			}
			var (
				size  int
				out   io.Writer = file
				zsize           = zfinfo.Size()
				bar   *progressbar.ProgressBar
			)
			if zsize > 0 {
				bar = progressbar.NewOptions64(zsize,
					// File transfers are usually slow, make sure we display the bar at 0%.
					progressbar.OptionSetRenderBlankState(true),
					// Display the filename.
					progressbar.OptionSetDescription(filename),
					progressbar.OptionClearOnFinish(),
				)
				out = io.MultiWriter(out, bar)
				zr.OnBlockDone = func(n int) {
					size += n
				}
			}

			// Uncompress.
			_, err = io.Copy(out, zr)
			if err != nil {
				return err
			}
			for _, c := range []io.Closer{zfile, file} {
				err := c.Close()
				if err != nil {
					return err
				}
			}

			if bar != nil {
				_ = bar.Clear()
				fmt.Printf("%s %d\n", zfilename, size)
			}
		}

		return nil
	}
}
