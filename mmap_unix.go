// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux darwin

package mmap

import (
	"fmt"
	"os"
	"runtime"

	syscall "golang.org/x/sys/unix"
)

func openFile(filename string, fl int) (*File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("mmap: could not open %q: %w", filename, err)
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()
	if size == 0 {
		return &File{flag: fl, fi: fi}, nil
	}
	if size < 0 {
		return nil, fmt.Errorf("mmap: file %q has negative size", filename)
	}
	if size != int64(int(size)) {
		return nil, fmt.Errorf("mmap: file %q is too large", filename)
	}

	prot := syscall.PROT_READ
	if fl&wFlag != 0 {
		prot |= syscall.PROT_WRITE
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), prot, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	r := &File{
		data: data,
		flag: fl,
		fi:   fi,
	}
	runtime.SetFinalizer(r, (*File).Close)
	return r, nil
}

// Sync commits the current contents of the file to stable storage.
func (f *File) Sync() error {
	if !f.wflag() {
		return errBadFD
	}
	return syscall.Msync(f.data, syscall.MS_SYNC)
}

// Close closes the memory-mapped file.
func (f *File) Close() error {
	if f.data == nil {
		return nil
	}
	data := f.data
	f.data = nil
	runtime.SetFinalizer(f, nil)
	return syscall.Munmap(data)
}
