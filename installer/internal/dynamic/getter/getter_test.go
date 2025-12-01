//go:build integration
// +build integration

package getter

import (
	"context"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	getterTestEnv     env.Environment
	getterClusterName string
)

const (
	getterNamespace = "getter-test-system"
)

func TestMain(m *testing.M) {
	getterClusterName = "getter-test"
	getterTestEnv = env.New()

	getterTestEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), getterClusterName),
		createGetterNamespace(getterNamespace),
		setupGetterTestData,
	).Finish(
		envfuncs.DestroyCluster(getterClusterName),
	)

	os.Exit(getterTestEnv.Run(m))
}

func createGetterNamespace(ns string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}

		return ctx, r.Create(ctx, namespace)
	}
}

func setupGetterTestData(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return ctx, err
	}

	// Create test ConfigMap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "getter-test-config",
			Namespace: getterNamespace,
		},
		Data: map[string]string{
			"test-key":     "test-value",
			"app-name":     "my-app",
			"database-url": "postgres://localhost:5432/mydb",
		},
	}

	if err := r.Create(ctx, testConfigMap); err != nil {
		return ctx, err
	}

	// Create test Secret
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "getter-test-secret",
			Namespace: getterNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": "admin",
			"password": "super-secret",
		},
	}

	if err := r.Create(ctx, testSecret); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func TestGetterE2E(t *testing.T) {
	feature := features.New("Getter E2E Tests").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Get ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			getter, err := NewGetter(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create getter: %v", err)
			}

			obj, err := getter.Get(ctx, GetOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ConfigMap",
				},
				Namespace: getterNamespace,
				Name:      "getter-test-config",
			})

			if err != nil {
				t.Fatalf("Failed to get ConfigMap: %v", err)
			}

			if obj.GetName() != "getter-test-config" {
				t.Errorf("Expected 'getter-test-config', got '%s'", obj.GetName())
			}

			if obj.GetNamespace() != getterNamespace {
				t.Errorf("Expected '%s', got '%s'", getterNamespace, obj.GetNamespace())
			}

			// Check data
			// data, found, err := obj.NestedStringMap("data")
			// if err != nil || !found {
			// 	t.Fatalf("Failed to get data field: %v", err)
			// }

			data, ok := obj.Object["data"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected data field to be a map[string]interface{}")
			}

			if data["test-key"] != "test-value" {
				t.Errorf("Expected 'test-value', got '%s'", data["test-key"])
			}

			t.Logf("ConfigMap get test passed: %s", obj.GetName())
			return ctx
		}).
		Assess("Get Secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			getter, err := NewGetter(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create getter: %v", err)
			}

			obj, err := getter.Get(ctx, GetOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Secret",
				},
				Namespace: getterNamespace,
				Name:      "getter-test-secret",
			})

			if err != nil {
				t.Fatalf("Failed to get Secret: %v", err)
			}

			if obj.GetName() != "getter-test-secret" {
				t.Errorf("Expected 'getter-test-secret', got '%s'", obj.GetName())
			}

			// Check data (base64 encoded in secrets)
			data, ok := obj.Object["data"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected data field to be a map[string]interface{}")
			}

			if len(data["username"].(string)) == 0 {
				t.Error("Expected username data to be present")
			}

			t.Logf("Secret get test passed: %s", obj.GetName())
			return ctx
		}).
		Assess("Get non-existent resource", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			getter, err := NewGetter(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create getter: %v", err)
			}

			_, err = getter.Get(ctx, GetOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ConfigMap",
				},
				Namespace: getterNamespace,
				Name:      "non-existent-configmap",
			})

			if err == nil {
				t.Fatal("Expected error for non-existent resource, got nil")
			}

			t.Logf("Non-existent resource test passed: got expected error: %v", err)
			return ctx
		}).
		Assess("Get cluster-scoped resource", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			getter, err := NewGetter(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create getter: %v", err)
			}

			// Try to get a namespace (cluster-scoped resource)
			obj, err := getter.Get(ctx, GetOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Namespace",
				},
				Name: getterNamespace,
			})

			if err != nil {
				t.Fatalf("Failed to get Namespace: %v", err)
			}

			if obj.GetName() != getterNamespace {
				t.Errorf("Expected '%s', got '%s'", getterNamespace, obj.GetName())
			}

			t.Logf("Cluster-scoped resource test passed: %s", obj.GetName())
			return ctx
		}).
		Feature()

	getterTestEnv.Test(t, feature)
}
