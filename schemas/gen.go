// +build generate

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	providers, err := listProviders("official")
	exitOnError(err)

	fmt.Println(providers)

}

type providerAttributes struct {
	Alias    string `json:"alias"`
	FullName string `json:"full-name"`
}

type provider struct {
	Attributes providerAttributes `json:"attributes"`
}

type registryResponse struct {
	Data []provider `json:"data"`
}

func listProviders(tier string) ([]provider, error) {
	// TODO will eventually need to paginate, for now official is 33 and partner is 82
	resp, err := http.Get(fmt.Sprintf("https://registry.terraform.io/v2/providers?page[size]=100&filter[tier]=%s", tier))
	if err != nil {
		return nil, err
	}

	var response registryResponse
	err = json.NewDecoder(resp.Body).Decode(&response)

	return response.Data, err
}
