// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package registry

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListProviders(t *testing.T) {
	client := NewClient()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v2/providers?page[size]=2&filter[tier]=official&page[number]=1" {
			w.Write([]byte(`{
  "data": [
    {
      "type": "providers",
      "id": "315",
      "attributes": {
        "alias": "azuread",
        "description": "Manage users, groups, service principals, and applications in Azure Active Directory using the Microsoft Graph API. This provider is maintained by the Azure providers team at HashiCorp.",
        "downloads": 46684900,
        "featured": false,
        "full-name": "hashicorp/azuread",
        "logo-url": "/images/providers/azure.svg?3",
        "name": "azuread",
        "namespace": "hashicorp",
        "owner-name": "",
        "robots-noindex": false,
        "source": "https://github.com/hashicorp/terraform-provider-azuread",
        "tier": "official",
        "unlisted": false,
        "warning": ""
      },
      "links": {
        "self": "/v2/providers/315"
      }
    },
    {
      "type": "providers",
      "id": "378",
      "attributes": {
        "alias": "http",
        "description": "Utility provider for interacting with generic HTTP servers as part of a Terraform configuration.",
        "downloads": 23888754,
        "featured": false,
        "full-name": "hashicorp/http",
        "logo-url": "/images/providers/hashicorp.svg",
        "name": "http",
        "namespace": "hashicorp",
        "owner-name": "",
        "robots-noindex": false,
        "source": "https://github.com/hashicorp/terraform-provider-http",
        "tier": "official",
        "unlisted": false,
        "warning": ""
      },
      "links": {
        "self": "/v2/providers/378"
      }
    }
  ],
  "links": {
    "first": "/v2/providers?filter%5Btier%5D=official&page%5Bnumber%5D=1&page%5Bsize%5D=2",
    "last": "/v2/providers?filter%5Btier%5D=official&page%5Bnumber%5D=2&page%5Bsize%5D=2",
    "next": "/v2/providers?filter%5Btier%5D=official&page%5Bnumber%5D=2&page%5Bsize%5D=2",
    "prev": null
  },
  "meta": {
    "pagination": {
      "page-size": 2,
      "current-page": 1,
      "next-page": 2,
      "prev-page": null,
      "total-pages": 2,
      "total-count": 3
    }
  }
}
`))
			return
		}
		if r.RequestURI == "/v2/providers?page[size]=2&filter[tier]=official&page[number]=2" {
			w.Write([]byte(`{
  "data": [
  	{
      "type": "providers",
      "id": "370",
      "attributes": {
        "alias": "tfe",
        "description": "Provision Terraform Cloud or Terraform Enterprise - with Terraform! Management of organizations, workspaces, teams, variables, run triggers, policy sets, and more. Maintained by the Terraform Cloud team at HashiCorp.",
        "downloads": 8555685,
        "featured": false,
        "full-name": "hashicorp/tfe",
        "logo-url": "/images/providers/terraform.svg?3",
        "name": "tfe",
        "namespace": "hashicorp",
        "owner-name": "",
        "robots-noindex": false,
        "source": "https://github.com/hashicorp/terraform-provider-tfe",
        "tier": "official",
        "unlisted": false,
        "warning": ""
      },
      "links": {
        "self": "/v2/providers/370"
      }
    }
  ],
  "links": {
    "first": "/v2/providers?filter%5Btier%5D=official&page%5Bnumber%5D=1&page%5Bsize%5D=2",
    "last": "/v2/providers?filter%5Btier%5D=official&page%5Bnumber%5D=2&page%5Bsize%5D=2",
    "next": null,
    "prev": "/v2/providers?filter%5Btier%5D=official&page%5Bnumber%5D=1&page%5Bsize%5D=2"
  },
  "meta": {
    "pagination": {
      "page-size": 2,
      "current-page": 1,
      "next-page": null,
      "prev-page": 1,
      "total-pages": 2,
      "total-count": 3
    }
  }
}
`))
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))
	client.BaseURL = srv.URL
	client.ProviderPageSize = 2
	t.Cleanup(srv.Close)

	providers, err := client.ListProviders("official")
	if err != nil {
		t.Fatal(err)
	}

	expectedProviders := []Provider{
		{
			ID: "315",
			Attributes: ProviderAttributes{
				Name:      "azuread",
				Namespace: "hashicorp",
			},
		},
		{
			ID: "378",
			Attributes: ProviderAttributes{
				Name:      "http",
				Namespace: "hashicorp",
			},
		},
		{
			ID: "370",
			Attributes: ProviderAttributes{
				Name:      "tfe",
				Namespace: "hashicorp",
			},
		},
	}
	if diff := cmp.Diff(expectedProviders, providers); diff != "" {
		t.Fatalf("unexpected providers: %s", diff)
	}
}

func TestGetLatestProviderVersion(t *testing.T) {
	client := NewClient()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/v2/providers/370/provider-versions/latest?include=provider-platforms" {
			w.Write([]byte(`{
  "data": {
    "type": "provider-versions",
    "id": "27307",
    "attributes": {
      "description": "terraform-provider-tfe",
      "downloads": 243313,
      "published-at": "2022-08-24T19:09:29Z",
      "tag": "v0.36.1",
      "version": "0.36.1"
    },
    "relationships": {
      "platforms": {
        "data": [
          {
            "type": "provider-platforms",
            "id": "287323"
          },
          {
            "type": "provider-platforms",
            "id": "287326"
          }
        ]
      }
    },
    "links": {
      "self": "/v2/provider-versions/27307"
    }
  },
  "included": [
    {
      "type": "provider-platforms",
      "id": "287327",
      "attributes": {
        "arch": "arm",
        "downloads": 237,
        "os": "freebsd"
      }
    },
    {
      "type": "provider-platforms",
      "id": "287324",
      "attributes": {
        "arch": "arm64",
        "downloads": 2481,
        "os": "darwin"
      }
    }
  ]
}`))
			return
		}
	}))
	client.BaseURL = srv.URL
	t.Cleanup(srv.Close)

	resp, err := client.GetLatestProviderVersion("370")
	if err != nil {
		t.Fatal(err)
	}

	expectedResponse := &ProviderVersionResponse{
		Data: ProviderVersionData{
			Attributes: ProviderVersionAttributes{
				Version: "0.36.1",
			},
		},
		Included: []Included{
			{
				Type: "provider-platforms",
				Attributes: IncludedAttributes{
					Arch: "arm",
					Os:   "freebsd",
				},
			},
			{
				Type: "provider-platforms",
				Attributes: IncludedAttributes{
					Arch: "arm64",
					Os:   "darwin",
				},
			},
		},
	}
	if diff := cmp.Diff(expectedResponse, resp); diff != "" {
		t.Fatalf("unexpected response: %s", diff)
	}
}
