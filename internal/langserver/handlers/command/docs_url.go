// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-ls/internal/utm"
)

func docsURL(ctx context.Context, rawURL, utmContent string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("utm_source", utm.UtmSource)
	if medium := utm.UtmMedium(ctx); medium != "" {
		q.Set("utm_medium", medium)
	}

	if utmContent != "" {
		q.Set("utm_content", utmContent)
	}

	u.RawQuery = q.Encode()

	return u, nil
}
