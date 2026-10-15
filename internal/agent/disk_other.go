//go:build !linux

package agent

func getLinuxDiskUsage(path string) (float64, int64) {
	// Fallback simulation for non-linux environments
	return 38.5, 45 * 1024 * 1024 * 1024
}
