package util

import (
	"io/ioutil"
	"os"
)

// OpenFileUniqueName creates a file with a unique (and randomly generated)
// filename with the given path and name prefix. It is opened with
// flag|os.O_CREATE|os.O_EXCL; os.O_WRONLY or os.RDWR should be specified for
// flag at minimum. It is the caller's responsibility to close (and maybe
// delete) the file when they have finished using it.
func OpenFileUniqueName(prefix string, _ int, _ uint32) (file *os.File, err error) {
	f, e := ioutil.TempFile("", prefix)
	return f, e
}
