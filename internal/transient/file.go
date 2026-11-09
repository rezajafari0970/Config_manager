package transient

import (
	"fmt"
	"os"
)

// The format-driver contract takes a reopenable path. On the Linux server,
// /proc/self/fd gives each reader a fresh offset while the original inode is
// already unlinked. Crashes cannot leave a named plaintext subscription behind.
func NewFile() (*os.File, string, error) {
	f, err := os.CreateTemp("", "config-fetch-")
	if err != nil {
		return nil, "", err
	}
	if err = os.Remove(f.Name()); err != nil {
		f.Close()
		return nil, "", err
	}
	return f, fmt.Sprintf("/proc/self/fd/%d", f.Fd()), nil
}
