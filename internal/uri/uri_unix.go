// +build !windows

package uri

// wrapPath is no-op for unix-style paths
func wrapPath(path string) string {
	return path
}
