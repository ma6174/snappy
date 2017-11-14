package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/golang/snappy"
)

type RWCounter struct {
	writer io.Writer
	reader io.Reader
	cr, cw int64
}

func (w *RWCounter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	w.cw += int64(n)
	return
}
func (w *RWCounter) Read(p []byte) (n int, err error) {
	n, err = w.reader.Read(p)
	w.cr += int64(n)
	return
}
func (w *RWCounter) CountR() int64 {
	return w.cr
}
func (w *RWCounter) CountW() int64 {
	return w.cw
}

func NewRWCounter(r io.Reader, w io.Writer) *RWCounter {
	return &RWCounter{
		reader: r,
		writer: w,
	}
}

func do(isDecompress bool, filename, suffix string, isToStdout bool, bufferSize int, isVerbose bool) (percentage, speed float64, err error) {
	var (
		input   io.Reader
		output  io.Writer
		outName string = "-"
		bar     *pb.ProgressBar
	)

	{
		bar = pb.New64(0)
		bar.ShowFinalTime = true
		bar.ShowPercent = true
		bar.ShowSpeed = true
		bar.ShowTimeLeft = true
		bar.ShowCounters = true
		bar.Output = os.Stderr
		bar.SetMaxWidth(80)
		bar.SetUnits(pb.U_BYTES)
	}

	if filename == "-" {
		input = os.Stdin
		output = os.Stdout
	} else {
		fi, err := os.Open(filename)
		if err != nil {
			return 0, 0, err
		}
		input = fi
		defer fi.Close()

		if isVerbose {
			info, err := os.Stat(filename)
			if err != nil {
				return 0, 0, err
			}
			bar.Total = info.Size()
		}

		if isToStdout {
			output = os.Stdout
		} else {
			if isDecompress {
				if !strings.HasSuffix(filename, suffix) {
					err = errors.New(fmt.Sprintf("file: %s not has suffix %s", filename, suffix))
					return 0, 0, err
				}
				outName = filename[:(len(filename) - len(suffix))]
			} else {
				outName = filename + suffix
			}
			fo, err := os.Create(outName)
			if err != nil {
				return 0, 0, err
			}
			output = fo
			defer fo.Close()
		}
	}

	if isVerbose {
		bar.Start()
		defer bar.Finish()
	}

	start := time.Now()
	rwc := NewRWCounter(bar.NewProxyReader(input), output)
	buf := make([]byte, bufferSize)
	if isDecompress {
		_, err = io.CopyBuffer(rwc, snappy.NewReader(rwc), buf)
	} else {
		_, err = io.CopyBuffer(snappy.NewWriter(rwc), rwc, buf)
	}
	useTime := time.Since(start).Seconds()
	if isDecompress {
		percentage = 1 - float64(rwc.CountR())/float64(rwc.CountW())
		speed = float64(rwc.CountW()) / 1024.0 / 1024.0 / useTime
	} else {
		percentage = 1 - float64(rwc.CountW())/float64(rwc.CountR())
		speed = float64(rwc.CountR()) / 1024.0 / 1024.0 / useTime
	}
	return
}

var helpMsg = `
SYNOPSIS
  snappy [-dcvh] [-s suffix] file [file [...]]
  cat file | snappy [-dcv]

OPTIONS:
`

func main() {
	var (
		isDecompress = flag.Bool("d", false, "Decompress")
		isToStdout   = flag.Bool("c", false, "Write  output  on standard output")
		isVerbose    = flag.Bool("v", false, "verbose display for name and percentage reduction and speed")
		Suffix       = flag.String("s", ".snappy", "output filename suffix")
		BufferSize   = flag.Int("b", 128, "buffer size for copy(KB)")
		files        []string
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprint(os.Stderr, helpMsg)
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()
	if flag.NArg() == 0 {
		files = []string{"-"}
	} else {
		files = flag.Args()
	}
	for _, filename := range files {
		percentage, speed, err := do(*isDecompress, filename, *Suffix, *isToStdout, *BufferSize*1024, *isVerbose)
		if err != nil {
			log.Printf("%s compress failed", err)
			continue
		}
		if *isVerbose {
			log.Printf("%s\t%.2f%%\t%.2fM/s\n", filename, percentage*100, speed)
		}
	}
}
