package util

import (
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/sys/unix"
)

// ListFiles lists non-dir files or first level files under the directories in the given path pattern
func ListFiles(directoryOrFilePattern string) ([]string, error) {
	inputList, gerr := filepath.Glob(directoryOrFilePattern)
	if gerr != nil {
		return nil, gerr
	}
	pathList := make([]string, 0, len(inputList)*2+10)
	for _, input := range inputList {
		stat, serr := os.Stat(input)
		if serr != nil {
			return nil, serr
		}
		if stat.IsDir() {
			fileList, rerr := os.ReadDir(input)
			if rerr != nil {
				return nil, rerr
			}
			for _, file := range fileList {
				pathList = append(pathList, filepath.Join(input, file.Name()))
			}
		} else {
			pathList = append(pathList, input)
		}
	}
	sort.Strings(pathList)
	return pathList, nil
}

// ReadFileAt reads full contents of a file in given directory
func ReadFileAt(dir *os.File, filename string) ([]byte, error) {
	fd, oerr := unix.Openat(int(dir.Fd()), filename, unix.O_RDONLY, 0o644)
	if oerr != nil {
		return nil, oerr
	}
	var stat unix.Stat_t
	if serr := unix.Fstat(fd, &stat); serr != nil {
		unix.Close(fd)
		return nil, serr
	}
	buf := make([]byte, stat.Size)
	n, rerr := unix.Read(fd, buf)
	if rerr != nil {
		unix.Close(fd)
		return nil, rerr
	}
	if n != len(buf) {
		buf = buf[:n]
	}
	unix.Close(fd)
	return buf, nil
}

// StatFileAt queries the stat of an existing file in given directory
func StatFileAt(dir *os.File, filename string) (unix.Stat_t, error) {
	var stat unix.Stat_t
	err := unix.Fstatat(int(dir.Fd()), filename, &stat, 0)
	return stat, err
}

// UnlinkFileAt unlinks an existing file in given directory
func UnlinkFileAt(dir *os.File, filename string) error {
	return unix.Unlinkat(int(dir.Fd()), filename, 0)
}

// WriteFileAt writes to a new file in given directory
func WriteFileAt(dir *os.File, filename string, data []byte, perm os.FileMode) error {
	fd, oerr := unix.Openat(int(dir.Fd()), filename, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC, uint32(perm))
	if oerr != nil {
		return oerr
	}
	_, werr := unix.Write(fd, data)
	unix.Close(fd)
	return werr
}
