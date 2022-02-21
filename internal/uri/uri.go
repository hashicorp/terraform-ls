package uri

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// FromPath creates a URI from OS-specific path per RFC 8089 "file" URI Scheme
func FromPath(rawPath string) string {
	// Cleaning up path trims any trailing separator
	// which then (in the context of URI below) complies
	// with RFC 3986 § 6.2.4 which is relevant in LSP.
	path := filepath.Clean(rawPath)

	// Convert any OS-specific separators to '/'
	path = filepath.ToSlash(path)

	volume := filepath.VolumeName(rawPath)
	if isWindowsDriveVolume(volume) {
		// Per RFC 8089 (Appendix F. Collected Nonstandard Rules)
		// file-absolute = "/" drive-letter path-absolute
		// i.e. paths with drive-letters (such as C:) are prepended
		// with an additional slash.
		path = "/" + path
	}

	u := &url.URL{
		Scheme: "file",
		Path:   path,
	}

	// Ensure that String() returns uniform escaped path at all times
	escapedPath := u.EscapedPath()
	if escapedPath != path {
		u.RawPath = escapedPath
	}

	return u.String()
}

// isWindowsDriveVolume returns true if the volume name has a drive letter.
// For example: C:\example.
func isWindowsDriveVolume(path string) bool {
	if len(path) < 2 {
		return false
	}
	return unicode.IsLetter(rune(path[0])) && path[1] == ':'
}

// IsURIValid checks whether uri is a valid URI per RFC 8089
func IsURIValid(uri string) bool {
	_, err := parseUri(uri)
	if err != nil {
		return false
	}

	return true
}

// PathFromURI extracts OS-specific path from an RFC 8089 "file" URI Scheme
func PathFromURI(rawUri string) (string, error) {
	uri, err := parseUri(rawUri)
	if err != nil {
		return "", err
	}

	// Convert '/' to any OS-specific separators
	osPath := filepath.FromSlash(uri.Path)

	// Upstream net/url parser prefers consistency and reusability
	// (e.g. in HTTP servers) which complies with
	// the Comparison Ladder as defined in § 6.2 of RFC 3968.
	// https://datatracker.ietf.org/doc/html/rfc3986#section-6.2
	//
	// Cleaning up path trims any trailing separator
	// which then still complies with RFC 3986 per § 6.2.4
	// which is relevant in LSP.
	osPath = filepath.Clean(osPath)

	trimmedOsPath := trimLeftPathSeparator(osPath)
	if strings.HasSuffix(filepath.VolumeName(trimmedOsPath), ":") {
		// Per RFC 8089 (Appendix F. Collected Nonstandard Rules)
		// file-absolute = "/" drive-letter path-absolute
		// i.e. paths with drive-letters (such as C:) are preprended
		// with an additional slash (which we converted to OS separator above)
		// which we trim here.
		// See also https://github.com/golang/go/issues/6027
		osPath = trimmedOsPath
	}

	return osPath, nil
}

// MustParseURI returns a normalized RFC 8089 URI.
// It will panic if rawUri is invalid.
//
// Use IsURIValid for checking validity upfront.
func MustParseURI(rawUri string) string {
	uri, err := parseUri(rawUri)
	if err != nil {
		panic(fmt.Sprintf("invalid URI: %s", uri))
	}

	return uri.String()
}

func trimLeftPathSeparator(s string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		return r == os.PathSeparator
	})
}

func MustPathFromURI(uri string) string {
	osPath, err := PathFromURI(uri)
	if err != nil {
		panic(fmt.Sprintf("invalid URI: %s", uri))
	}
	return osPath
}

func parseUri(rawUri string) (*url.URL, error) {
	uri, err := url.ParseRequestURI(rawUri)
	if err != nil {
		return nil, err
	}

	if uri.Scheme != "file" {
		return nil, fmt.Errorf("unexpected scheme %q in URI %q",
			uri.Scheme, rawUri)
	}

	// Upstream net/url parser prefers consistency and reusability
	// (e.g. in HTTP servers) which complies with
	// the Comparison Ladder as defined in § 6.2 of RFC 3968.
	// https://datatracker.ietf.org/doc/html/rfc3986#section-6.2
	// Here we essentially just implement § 6.2.4
	// as it is relevant in LSP (which uses the file scheme).
	uri.Path = strings.TrimSuffix(uri.Path, "/")

	// Upstream net/url parser (correctly) escapes only
	// non-ASCII characters as per § 2.1 of RFC 3986.
	// https://datatracker.ietf.org/doc/html/rfc3986#section-2.1
	// Unfortunately VSCode effectively violates that section
	// by escaping ASCII characters such as colon.
	// See https://github.com/microsoft/vscode/issues/75027
	//
	// To account for this we reset RawPath which would
	// otherwise be used by String() to effectively enforce
	// clean re-escaping of the (unescaped) Path.
	uri.RawPath = ""

	return uri, nil
}
