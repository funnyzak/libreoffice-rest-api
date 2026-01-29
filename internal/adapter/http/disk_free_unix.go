//go:build !windows

package http

import "golang.org/x/sys/unix"

func diskFreeGB(path string) (int, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	free := stat.Bavail * uint64(stat.Bsize)
	return int(free / (1024 * 1024 * 1024)), nil
}
