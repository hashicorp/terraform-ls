// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mdplain

import (
	"regexp"
)

type replacement struct {
	re  *regexp.Regexp
	sub string
}

var replacements = []replacement{
	// rules heavily inspired by: https://github.com/stiang/remove-markdown/blob/master/index.js
	// back references were removed

	// Header
	{regexp.MustCompile(`\n={2,}`), "\n"},
	// Fenced codeblocks
	{regexp.MustCompile(`~{3}.*\n`), ""},
	// Strikethrough
	{regexp.MustCompile("~~"), ""},
	// Fenced codeblocks
	{regexp.MustCompile("`{3}.*\\n"), ""},
	// Remove HTML tags
	{regexp.MustCompile(`<[^>]*>`), ""},
	// Remove setext-style headers
	{regexp.MustCompile(`^[=\-]{2,}\s*$`), ""},
	// Remove footnotes?
	{regexp.MustCompile(`\[\^.+?\](\: .*?$)?`), ""},
	{regexp.MustCompile(`\s{0,2}\[.*?\]: .*?$`), ""},
	// Remove images
	{regexp.MustCompile(`\!\[(.*?)\][\[\(].*?[\]\)]`), "$1"},
	// Remove inline links
	{regexp.MustCompile(`\[(.*?)\][\[\(].*?[\]\)]`), "$1"},
	// Remove blockquotes
	{regexp.MustCompile(`^\s{0,3}>\s?`), ""},
	// Remove reference-style links?
	{regexp.MustCompile(`^\s{1,2}\[(.*?)\]: (\S+)( ".*?")?\s*$`), ""},
	// Remove atx-style headers
	{regexp.MustCompile(`^(\n)?\s{0,}#{1,6}\s+| {0,}(\n)?\s{0,}#{0,} {0,}(\n)?\s{0,}$`), "$1$2$3"},
	// Remove emphasis (repeat the line to remove double emphasis)
	{regexp.MustCompile(`([*_]{1,3})([^\t\n\f\r *_].*?[^\t\n\f\r *_]{0,1})([*_]{1,3})`), "$2"},
	{regexp.MustCompile(`([*_]{1,3})([^\t\n\f\r *_].*?[^\t\n\f\r *_]{0,1})([*_]{1,3})`), "$2"},
	// Remove code blocks
	{regexp.MustCompile("(`{3,})(.*?)(`{3,})"), "$2"},
	// Remove inline code
	{regexp.MustCompile("`(.+?)`"), "$1"},
	// Replace two or more newlines with exactly two? Not entirely sure this belongs here...
	{regexp.MustCompile(`\n{2,}`), "\n\n"},
}

// Clean runs a VERY naive cleanup of markdown text to make it more palatable as plain text.
func Clean(markdown string) string {
	// TODO: maybe use https://github.com/russross/blackfriday/tree/v2, write custom renderer or
	// generate HTML then process that to plaintext using https://github.com/jaytaylor/html2text

	for _, r := range replacements {
		markdown = r.re.ReplaceAllString(markdown, r.sub)
	}

	return string(markdown)
}
