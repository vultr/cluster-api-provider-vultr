package scope

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateVultrClient(apiKey string) (*govultr.Client, error) {
	if apiKey == "" {
		return nil, errors.New("VULTR_API_KEY is required")
	}
	config := &oauth2.Config{}
	ctx := context.Background()
	tokenSource := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, tokenSource))

	return vultrClient, nil
}

func addCredentialsFinalizer(ctx context.Context, v VultrAPIClients, credentialsRef corev1.SecretReference, defaultNamespace, finalizer string) error {
	secret, err := getCredentials(ctx, v, credentialsRef, defaultNamespace)
	if err != nil {
		return err
	}

	controllerutil.AddFinalizer(secret, finalizer)
	if err := v.Update(ctx, secret); err != nil {
		return fmt.Errorf("add finalizer to credentials secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}
	return nil
}

// getCredentials fetches the secret referenced by credentialsRef.
func getCredentials(ctx context.Context, v VultrAPIClients, credentialsRef corev1.SecretReference, defaultNamespace string) (*corev1.Secret, error) {
	namespace := credentialsRef.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: credentialsRef.Name}
	if err := v.Get(ctx, key, secret); err != nil {
		return nil, fmt.Errorf("get credentials secret %s/%s: %w", namespace, credentialsRef.Name, err)
	}
	return secret, nil
}

// toFinalizer converts an object into a valid finalizer key representation
func toFinalizer(obj client.Object) string {
	var (
		gvk       = obj.GetObjectKind().GroupVersionKind()
		group     = gvk.Group
		kind      = strings.ToLower(gvk.Kind)
		namespace = obj.GetNamespace()
		name      = obj.GetName()
	)
	if namespace == "" {
		return fmt.Sprintf("%s.%s/%s", kind, group, name)
	}
	return fmt.Sprintf("%s.%s/%s.%s", kind, group, namespace, name)
}
