package scope

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/vultr/cluster-api-provider-vultr/api/v1beta2"
)

// providerIDScheme is the scheme used in providerIDs for every Vultr resource
// kind (vultr://<id>); the Vultr cloud controller manager expects the same
// scheme for instances and bare metal servers.
const providerIDScheme = "vultr"

// ProviderIDToResourceID returns the Vultr resource id encoded in a providerID
// of the form vultr://<id>, or an empty string if the providerID is malformed.
func ProviderIDToResourceID(providerID string) string {
	split := strings.Split(providerID, "://")
	if len(split) != 2 { //nolint
		return ""
	}

	if split[0] != providerIDScheme {
		return ""
	}
	return split[1]
}

// ResourceIDToProviderID builds the providerID for a Vultr resource id.
func ResourceIDToProviderID(resourceID string) string {
	return providerIDScheme + "://" + resourceID
}

// MachineRole returns the tag role value for a Machine.
func MachineRole(machine *clusterv1.Machine) string {
	if util.IsControlPlaneMachine(machine) {
		return infrav1.ControlPlaneRoleTagValue
	}
	return infrav1.WorkerRoleTagValue
}

// GetBootstrapData reads the bootstrap data secret referenced by the Machine.
func GetBootstrapData(ctx context.Context, c client.Client, log logr.Logger, machine *clusterv1.Machine, namespace string) (string, error) {
	if machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap data secret reference is nil")
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secretName := *machine.Spec.Bootstrap.DataSecretName
	key := types.NamespacedName{Namespace: namespace, Name: secretName}
	log.Info("Attempting to retrieve bootstrap data secret", "namespace", key.Namespace, "name", key.Name)

	secret := &corev1.Secret{}
	if err := c.Get(ctx, key, secret); err != nil {
		log.Error(err, "Failed to retrieve bootstrap data secret", "namespace", key.Namespace, "name", key.Name)
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret %s/%s", key.Namespace, key.Name)
	}

	value, ok := secret.Data["value"]
	if !ok {
		log.Info("Bootstrap data secret missing 'value' key")
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return string(value), nil
}
