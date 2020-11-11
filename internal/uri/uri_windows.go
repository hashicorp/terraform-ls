package uri

// wrapPath prepends Windows-style paths (C:\path)
// with an additional slash to account for an empty hostname
// in a valid file-scheme URI per RFC 8089
func wrapPath(path string) string {
	return "/" + path
}
