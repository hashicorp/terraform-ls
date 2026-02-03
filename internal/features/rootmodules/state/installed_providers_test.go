// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	globalState "github.com/hashicorp/terraform-ls/internal/state"
)

func TestInstalledProviders(t *testing.T) {
	testCases := []struct {
		first, second InstalledProviders
		expectEqual   bool
	}{
		{
			InstalledProviders{},
			InstalledProviders{},
			true,
		},
		{
			InstalledProviders{
				globalState.NewBuiltInProvider("terraform"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				globalState.NewBuiltInProvider("terraform"): version.Must(version.NewVersion("1.0")),
			},
			true,
		},
		{
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				globalState.NewDefaultProvider("bar"): version.Must(version.NewVersion("1.0")),
			},
			false,
		},
		{
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.1")),
			},
			false,
		},
		{
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
				globalState.NewDefaultProvider("bar"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			false,
		},
		{
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				globalState.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
				globalState.NewDefaultProvider("bar"): version.Must(version.NewVersion("1.0")),
			},
			false,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			equals := tc.first.Equals(tc.second)
			if tc.expectEqual != equals {
				if tc.expectEqual {
					t.Fatalf("expected requirements to be equal\nfirst: %#v\nsecond: %#v", tc.first, tc.second)
				}
				t.Fatalf("expected requirements to mismatch\nfirst: %#v\nsecond: %#v", tc.first, tc.second)
			}
		})
	}
}
