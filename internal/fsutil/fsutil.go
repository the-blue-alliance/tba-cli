// Package fsutil holds the filesystem helpers more than one package needs.
//
// It exists because writing a file safely is fiddly enough to get subtly
// wrong four separate times: the temporary file has to live in the target
// directory (a rename across filesystems is not atomic), it has to be given
// the mode the finished file should have rather than the private mode
// os.CreateTemp hands out, and every failure along the way has to take the
// temporary file with it.
package fsutil

import (
	"os"
	"path/filepath"
)

// renameFile is os.Rename, replaceable so that a test can see what a failed
// rename leaves behind.
var renameFile = os.Rename

// WriteFileAtomic writes data to path through a temporary file in the same
// directory and a rename, so a concurrent reader sees either the old contents
// or the new ones and never a half-written file. The finished file has mode
// perm. Nothing is left behind if any step fails.
//
// The directory must already exist: creating it is a decision about layout,
// which belongs to the caller.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := WriteTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp", data, perm)
	if err != nil {
		return err
	}
	if err := renameFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// WriteTemp writes data to a new temporary file in dir with mode perm and
// returns its name, for callers that stage several files and rename them into
// place together. The caller owns the file from then on: it has to be renamed
// or removed.
//
// pattern is an os.CreateTemp pattern; the "*" is where the random part goes.
func WriteTemp(dir, pattern string, data []byte, perm os.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	abandon := func(err error) (string, error) {
		_ = os.Remove(name)
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return abandon(err)
	}
	if err := f.Close(); err != nil {
		return abandon(err)
	}
	// os.CreateTemp makes a 0600 file. Saying the mode explicitly means a
	// file's permissions come from what it is for, not from which helper
	// happened to create it.
	if err := os.Chmod(name, perm); err != nil {
		return abandon(err)
	}
	return name, nil
}
