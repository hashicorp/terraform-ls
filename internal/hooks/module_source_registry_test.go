package hooks

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/terraform-ls/internal/state"
	"github.com/zclconf/go-cty/cty"
)

const responseAWS = `{
	"hits": [
		{
			"full-name": "terraform-aws-modules/vpc/aws",
			"description": "Terraform module which creates VPC resources on AWS",
			"objectID": "modules:23"
		},
		{
			"full-name": "terraform-aws-modules/eks/aws",
			"description": "Terraform module to create an Elastic Kubernetes (EKS) cluster and associated resources",
			"objectID": "modules:1143"
		}
	],
	"nbHits": 10200,
	"page": 0,
	"nbPages": 100,
	"hitsPerPage": 2,
	"exhaustiveNbHits": true,
	"exhaustiveTypo": true,
	"query": "aws",
	"params": "attributesToRetrieve=%5B%22full-name%22%2C%22description%22%5D&hitsPerPage=2&query=aws",
	"renderingContent": {},
	"processingTimeMS": 1,
	"processingTimingsMS": {}
}`

const responseEmpty = `{
	"hits": [],
	"nbHits": 0,
	"page": 0,
	"nbPages": 0,
	"hitsPerPage": 2,
	"exhaustiveNbHits": true,
	"exhaustiveTypo": true,
	"query": "foo",
	"params": "attributesToRetrieve=%5B%22full-name%22%2C%22description%22%5D&hitsPerPage=2&query=foo",
	"renderingContent": {},
	"processingTimeMS": 1
}`

const responseErr = `{
	"message": "Invalid Application-ID or API key",
	"status": 403
}`

type testRequester struct {
	client *http.Client
}

func (r *testRequester) Request(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}

func TestHooks_RegistryModuleSources(t *testing.T) {
	ctx := context.Background()

	s, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	searchClient := buildSearchClientMock(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/1/indexes/tf-registry%3Aprod%3Amodules/query" {
			b, _ := io.ReadAll(r.Body)

			if strings.Contains(string(b), "query=aws") {
				w.Write([]byte(responseAWS))
				return
			} else if strings.Contains(string(b), "query=err") {
				http.Error(w, responseErr, http.StatusForbidden)
				return
			}

			w.Write([]byte(responseEmpty))
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))

	h := &Hooks{
		ModStore:      s.Modules,
		AlgoliaClient: searchClient,
	}

	tests := []struct {
		name    string
		value   cty.Value
		want    []decoder.Candidate
		wantErr bool
	}{
		{
			"simple search",
			cty.StringVal("aws"),
			[]decoder.Candidate{
				{
					Label:         `"terraform-aws-modules/vpc/aws"`,
					Detail:        "registry",
					Kind:          lang.StringCandidateKind,
					Description:   lang.PlainText("Terraform module which creates VPC resources on AWS"),
					RawInsertText: `"terraform-aws-modules/vpc/aws"`,
				},
				{
					Label:         `"terraform-aws-modules/eks/aws"`,
					Detail:        "registry",
					Kind:          lang.StringCandidateKind,
					Description:   lang.PlainText("Terraform module to create an Elastic Kubernetes (EKS) cluster and associated resources"),
					RawInsertText: `"terraform-aws-modules/eks/aws"`,
				},
			},
			false,
		},
		{
			"empty result",
			cty.StringVal("foo"),
			[]decoder.Candidate{},
			false,
		},
		{
			"auth error",
			cty.StringVal("err"),
			[]decoder.Candidate{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates, err := h.RegistryModuleSources(ctx, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("Hooks.RegistryModuleSources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, candidates); diff != "" {
				t.Fatalf("mismatched candidates: %s", diff)
			}
		})
	}
}

func TestHooks_RegistryModuleSourcesCtxCancel(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithTimeout(ctx, 50*time.Millisecond)
	t.Cleanup(cancelFunc)

	s, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	searchClient := buildSearchClientMock(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		http.Error(w, fmt.Sprintf("unexpected request: %q", r.RequestURI), 400)
	}))

	h := &Hooks{
		ModStore:      s.Modules,
		AlgoliaClient: searchClient,
	}

	_, err = h.RegistryModuleSources(ctx, cty.StringVal("aws"))
	e, ok := err.(net.Error)
	if !ok {
		t.Fatalf("expected error, got %#v", err)
	}

	if !strings.Contains(e.Error(), "context deadline exceeded") {
		t.Fatalf("expected error with: %q, given: %q", "context deadline exceeded", e.Error())
	}
}

func buildSearchClientMock(t *testing.T, handler http.HandlerFunc) *search.Client {
	searchServer := httptest.NewTLSServer(handler)
	t.Cleanup(searchServer.Close)

	// Algolia requires hosts to be without a protocol and always assumes https
	u, err := url.Parse(searchServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	searchClient := search.NewClientWithConfig(search.Configuration{
		Hosts: []string{u.Host},
		// We need to disable certificate checking here, because of the self signed cert
		Requester: &testRequester{
			client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			},
		},
	})

	return searchClient
}
