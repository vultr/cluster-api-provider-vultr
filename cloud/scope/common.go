package scope

import (
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
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, tokenSource))

	return vultrClient, nil
}
