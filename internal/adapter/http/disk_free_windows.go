//go:build windows

package http

import (
	"golang.org/x/sys/windows"
)

func diskFreeGB(path string) (int, error) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var freeBytesAvailable uint64
	if err := windows.GetDiskFreeSpaceEx(ptr, &freeBytesAvailable, nil, nil); err != nil {
		return 0, err
	}
	return int(freeBytesAvailable / (1024 * 1024 * 1024)), nil
}
