package scope

import (
	"github.com/vultr/cluster-api-provider-vultr/cloud/vultrerrors"

	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

func CreateVultrClient() (*govultr.Client, error) {
	apiKey := os.Getenv("VULTR_API_KEY")
	if apiKey == "" {
		return nil, errors.New("VULTR_API_KEY is required")
	}
	config := &oauth2.Config{}
	ctx := context.Background()
	tokenSource := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	httpClient := oauth2.NewClient(ctx, tokenSource)
	httpClient.Transport = vultrerrors.Transport{Next: httpClient.Transport}
	vultrClient := govultr.NewClient(httpClient)
	vultrClient.SetUserAgent("vultr-cluster-api")

	return vultrClient, nil
}

func MachineOnly() bool {
	return os.Getenv("VULTR_MACHINE_ONLY") == "true"
}
