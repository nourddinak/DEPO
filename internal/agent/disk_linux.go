//go:build linux

package agent

import (
	"golang.org/x/sys/unix"
)

func getLinuxDiskUsage(path string) (float64, int64) {
	if path == "" {
		path = "/"
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0.0, 0
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)

	if totalBytes == 0 {
		return 0.0, 0
	}

	usedBytes := totalBytes - freeBytes
	diskPct := (float64(usedBytes) / float64(totalBytes)) * 100.0

	return diskPct, int64(freeBytes)
}
