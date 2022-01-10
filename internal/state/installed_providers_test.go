package state

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
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
				tfaddr.NewBuiltInProvider("terraform"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				tfaddr.NewBuiltInProvider("terraform"): version.Must(version.NewVersion("1.0")),
			},
			true,
		},
		{
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				tfaddr.NewDefaultProvider("bar"): version.Must(version.NewVersion("1.0")),
			},
			false,
		},
		{
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.1")),
			},
			false,
		},
		{
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
				tfaddr.NewDefaultProvider("bar"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			false,
		},
		{
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
			},
			InstalledProviders{
				tfaddr.NewDefaultProvider("foo"): version.Must(version.NewVersion("1.0")),
				tfaddr.NewDefaultProvider("bar"): version.Must(version.NewVersion("1.0")),
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
