package shred

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func EmptyDir(path string) error {
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	_, err = fd.Readdirnames(1)
	if err == io.EOF {
		return nil
	}
	return errors.New("not an empty directory")
}

// Remove removes the named file or directory after overwriting its contents
// and name. This may or may not, depending on the filesystem and the
// underlying block device, unrecoverably erase the file and its name.
// The error is NOT necessarily of type *PathError.
func Remove(name string) error {
	finfo, err := os.Stat(name)
	if err != nil {
		return err
	}
	if finfo.IsDir() {
		if err := EmptyDir(name); err != nil {
			return err
		}
	} else {
		f, err := os.OpenFile(name, os.O_WRONLY, 0)
		if err != nil {
			return err
		}

		if _, err := io.CopyN(f, rand.Reader, finfo.Size()); err != nil {
			return err
		}
	}

	l := (len(finfo.Name()) + 1) / 2
	newBaseNameBytes := make([]byte, l)
	if _, err := rand.Read(newBaseNameBytes); err != nil {
		return err
	}
	newBaseName := hex.EncodeToString(newBaseNameBytes)
	newName := filepath.Join(filepath.Dir(name), newBaseName)
	if err := os.Rename(name, newName); err != nil {
		return err
	}
	return os.Remove(newName)
}

// RemoveAll removes path and any children it contains. It removes everything
// it can but returns the first error it encounters.  If the path does not
// exist, RemoveAll returns nil (no error).
func RemoveAll(path string) error {
	// Copyright 2009 The Go Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.

	// Simple case: if Remove works, we're done.
	err := Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	// Otherwise, is this a directory we need to recurse into?
	dir, serr := os.Lstat(path)
	if serr != nil {
		if serr, ok := serr.(*os.PathError); ok && (os.IsNotExist(serr.Err) || serr.Err == syscall.ENOTDIR) {
			return nil
		}
		return serr
	}
	if !dir.IsDir() {
		// Not a directory; return the error from Remove.
		return err
	}

	// Directory.
	fd, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Race. It was deleted between the Lstat and Open.
			// Return nil per RemoveAll's docs.
			return nil
		}
		return err
	}

	// Remove contents & return first error.
	err = nil
	for {
		names, err1 := fd.Readdirnames(100)
		for _, name := range names {
			err1 := RemoveAll(path + string(os.PathSeparator) + name)
			if err == nil {
				err = err1
			}
		}
		if err1 == io.EOF {
			break
		}
		// If Readdirnames returned an error, use it.
		if err == nil {
			err = err1
		}
		if len(names) == 0 {
			break
		}
	}

	// Close directory, because windows won't remove opened directory.
	fd.Close()

	// Remove directory.
	err1 := Remove(path)
	if err1 == nil || os.IsNotExist(err1) {
		return nil
	}
	if err == nil {
		err = err1
	}
	return err
}
