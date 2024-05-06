package scope

import (
	"context"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ClusterScopeParams defines the input parameters used to create a new scope.
type ClusterScopeParams struct {
	VultrAPIClients
	Client       client.Client
	Logger       logr.Logger
	Cluster      *clusterv1.Cluster
	VultrCluster *infrav1.VultrCluster
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(ctx context.Context, apiKey string, params ClusterScopeParams) (*ClusterScope, error) {

	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a ClusterScope")
	}

	if params.VultrCluster == nil {
		return nil, errors.New("VultrCluster is required when creating a ClusterScope")
	}

	if apiKey == "" {
		return nil, errors.New("environment variable VULTR_API_KEY is required")
	}

	config := &oauth2.Config{}
	tokenSource := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, tokenSource))

	helper, err := patch.NewHelper(params.VultrCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &ClusterScope{
		client:          params.Client,
		Cluster:         params.Cluster,
		VultrCluster:    params.VultrCluster,
		VultrAPIClients: params.VultrAPIClients,
		patchHelper:     helper,
		vultrClient:     vultrClient,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	VultrAPIClients
	Cluster      *clusterv1.Cluster
	VultrCluster *infrav1.VultrCluster
	vultrClient  *govultr.Client
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.VultrCluster)
}

func (s *ClusterScope) AddFinalizer(ctx context.Context) error {
	if controllerutil.AddFinalizer(s.VultrCluster, infrav1.GroupVersion.String()) {
		return s.Close()
	}

	return nil
}

func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.VultrCluster)
}
