// config-compat applies qualified structural rules; it never certifies health.
package main

import (
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"io"
	"os"
)

func main() {
	b, e := io.ReadAll(io.LimitReader(os.Stdin, 2*1024*1024+1))
	if e != nil || len(b) > 2*1024*1024 {
		fmt.Fprintln(os.Stderr, "input_size_or_read_error")
		os.Exit(2)
	}
	out, _, e := compat.ApplyJSON(b)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	os.Stdout.Write(out)
}
