package cmds

import (
	"flag"
	"fmt"
	"io"
	"os"

	"code.cloudfoundry.org/bytefmt"
	"github.com/schollz/progressbar/v2"

	"github.com/pierrec/lz4"
	"github.com/pierrec/lz4/internal/cmdflag"
)

// Compress compresses a set of files or from stdin to stdout.
func Compress(fs *flag.FlagSet) cmdflag.Handler {
	var blockMaxSize string
	fs.StringVar(&blockMaxSize, "size", "4M", "block max size [64K,256K,1M,4M]")
	var blockChecksum bool
	fs.BoolVar(&blockChecksum, "bc", false, "enable block checksum")
	var streamChecksum bool
	fs.BoolVar(&streamChecksum, "sc", false, "disable stream checksum")
	var level int
	fs.IntVar(&level, "l", 0, "compression level (0=fastest)")

	return func(args ...string) error {
		sz, err := bytefmt.ToBytes(blockMaxSize)
		if err != nil {
			return err
		}

		zw := lz4.NewWriter(nil)
		zw.Header = lz4.Header{
			BlockChecksum:    blockChecksum,
			BlockMaxSize:     int(sz),
			NoChecksum:       streamChecksum,
			CompressionLevel: level,
		}

		// Use stdin/stdout if no file provided.
		if len(args) == 0 {
			zw.Reset(os.Stdout)
			_, err := io.Copy(zw, os.Stdin)
			if err != nil {
				return err
			}
			return zw.Close()
		}

		for _, filename := range args {
			// Input file.
			file, err := os.Open(filename)
			if err != nil {
				return err
			}
			finfo, err := file.Stat()
			if err != nil {
				return err
			}
			mode := finfo.Mode() // use the same mode for the output file

			// Accumulate compressed bytes num.
			var (
				zsize int
				size  = finfo.Size()
			)
			if size > 0 {
				// Progress bar setup.
				numBlocks := int(size) / zw.Header.BlockMaxSize
				bar := progressbar.NewOptions(numBlocks,
					// File transfers are usually slow, make sure we display the bar at 0%.
					progressbar.OptionSetRenderBlankState(true),
					// Display the filename.
					progressbar.OptionSetDescription(filename),
					progressbar.OptionClearOnFinish(),
				)
				zw.OnBlockDone = func(n int) {
					_ = bar.Add(1)
					zsize += n
				}
			}

			// Output file.
			zfilename := fmt.Sprintf("%s%s", filename, lz4.Extension)
			zfile, err := os.OpenFile(zfilename, os.O_CREATE|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			zw.Reset(zfile)

			// Compress.
			_, err = io.Copy(zw, file)
			if err != nil {
				return err
			}
			for _, c := range []io.Closer{zw, zfile} {
				err := c.Close()
				if err != nil {
					return err
				}
			}

			if size > 0 {
				fmt.Printf("%s %.02f%%\n", zfilename, float64(zsize)*100/float64(size))
			}
		}

		return nil
	}
}
