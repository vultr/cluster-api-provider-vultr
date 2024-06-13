package scope

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1"
	"github.com/vultr/govultr/v3"
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

	// config := &oauth2.Config{}
	// tokenSource := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	// vultrClient := govultr.NewClient(oauth2.NewClient(ctx, tokenSource))

	vultrClient, err := CreateVultrClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vultr Client: %w", err)
	}

	helper, err := patch.NewHelper(params.VultrCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &ClusterScope{
		client:       params.Client,
		Cluster:      params.Cluster,
		VultrCluster: params.VultrCluster,
		//VultrAPIClients: params.VultrAPIClients,
		patchHelper: helper,
		vultrClient: vultrClient,
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

func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.VultrCluster)
}

func (s *ClusterScope) AddFinalizer(ctx context.Context) error {
	if controllerutil.AddFinalizer(s.VultrCluster, infrav1.GroupVersion.String()) {
		return s.Close()
	}

	return nil
}

// APIServerLoadbalancers get the VultrCluster Spec Network APIServerLoadbalancers.
func (s *ClusterScope) APIServerLoadbalancers() *infrav1.VultrLoadBalancer {
	return &s.VultrCluster.Spec.Network.APIServerLoadbalancers
}

func (s *ClusterScope) AddCredentialsRefFinalizer(ctx context.Context) error {
	if s.VultrCluster.Spec.CredentialsRef == nil {
		return nil
	}

	return addCredentialsFinalizer(ctx, s.VultrAPIClients, *s.VultrCluster.Spec.CredentialsRef, s.VultrCluster.GetNamespace(), toFinalizer(s.VultrCluster))
}

// APIServerLoadbalancersRef get the DOCluster status Network APIServerLoadbalancersRef.
func (s *ClusterScope) APIServerLoadbalancersRef() *infrav1.VultrResourceReference {
	return &s.VultrCluster.Status.Network.APIServerLoadbalancersRef
}

// UID returns the cluster UID.
func (s *ClusterScope) UID() string {
	return string(s.Cluster.UID)
}

// Region returns the cluster region.
func (s *ClusterScope) Region() string {
	return s.VultrCluster.Spec.Region
}

// Name returns the cluster name.
func (s *ClusterScope) Name() string {
	return s.Cluster.GetName()
}

// SetReady sets the DOCluster Ready Status.
func (s *ClusterScope) SetReady() {
	s.VultrCluster.Status.Ready = true
}

// SetControlPlaneEndpoint sets the VultrCluster status APIEndpoints.
func (s *ClusterScope) SetControlPlaneEndpoint(apiEndpoint clusterv1.APIEndpoint) {
	s.VultrCluster.Spec.ControlPlaneEndpoint = apiEndpoint
}
