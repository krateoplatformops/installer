package resolvers

import (
	"context"
	"encoding/base64"
	"fmt"

	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/krateoplatformops/installer/internal/dynamic"
)

// func GetSecret(ctx context.Context, kube client.Client, secretKeySelector rtv1.SecretKeySelector) (string, error) {
// 	secret := &corev1.Secret{}
// 	if err := kube.Get(ctx, types.NamespacedName{
// 		Name:      secretKeySelector.Name,
// 		Namespace: secretKeySelector.Namespace,
// 	}, secret); err != nil {
// 		return "", err
// 	}

// 	return string(secret.Data[secretKeySelector.Key]), nil
// }

func GetSecret(ctx context.Context, dyn dynamic.Getter, secretKeySelector rtv1.SecretKeySelector) (string, error) {
	uns, err := dyn.Get(ctx, dynamic.GetOptions{
		GVK:       corev1.SchemeGroupVersion.WithKind("Secret"),
		Namespace: secretKeySelector.Namespace,
		Name:      secretKeySelector.Name,
	})
	if err != nil {
		return "", err
	}
	data, ok, err := unstructured.NestedMap(uns.Object, "data")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("data field not found in secret %s/%s", secretKeySelector.Namespace, secretKeySelector.Name)
	}

	sec, ok := data[secretKeySelector.Key].(string)
	if !ok {
		return "", fmt.Errorf("key %s is not a string", secretKeySelector.Key)
	}
	bsec, err := base64.StdEncoding.DecodeString(sec)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 string: %w", err)
	}

	sec = string(bsec)
	return sec, nil

}
