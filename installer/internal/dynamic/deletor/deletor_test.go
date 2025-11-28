//go:build integration
// +build integration

package deletor

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	deletorTestEnv     env.Environment
	deletorClusterName string
)

const (
	deletorNamespace = "deletor-test-system"
)

func TestMain(m *testing.M) {
	deletorClusterName = "deletor-test"
	deletorTestEnv = env.New()

	deletorTestEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), deletorClusterName),
		createDeletorNamespace(deletorNamespace),
	).Finish(
		envfuncs.DestroyCluster(deletorClusterName),
	)

	os.Exit(deletorTestEnv.Run(m))
}

func createDeletorNamespace(ns string) env.Func {
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

func createTestResource(ctx context.Context, cfg *envconf.Config, name string) error {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}

	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: deletorNamespace,
		},
		Data: map[string]string{
			"test-key": "test-value",
		},
	}

	return r.Create(ctx, testConfigMap)
}

func TestDeletorE2E(t *testing.T) {
	feature := features.New("Deletor E2E Tests").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Delete ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create a test ConfigMap first
			err := createTestResource(ctx, cfg, "deletor-test-cm")
			if err != nil {
				t.Fatalf("Failed to create test ConfigMap: %v", err)
			}

			deletor, err := NewDeletor(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create deletor: %v", err)
			}

			err = deletor.Delete(ctx, DeleteOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ConfigMap",
				},
				Namespace: deletorNamespace,
				Name:      "deletor-test-cm",
			})

			if err != nil {
				t.Fatalf("Failed to delete ConfigMap: %v", err)
			}

			// Wait for deletion to propagate
			time.Sleep(2 * time.Second)

			// Verify the ConfigMap was deleted
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			err = r.Get(ctx, "deletor-test-cm", deletorNamespace, &cm)
			if err == nil {
				t.Fatal("Expected ConfigMap to be deleted, but it still exists")
			}

			if !errors.IsNotFound(err) {
				t.Fatalf("Expected NotFound error, got: %v", err)
			}

			t.Logf("ConfigMap deletion test passed")
			return ctx
		}).
		Assess("Delete Secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create a test Secret first
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			testSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deletor-test-secret",
					Namespace: deletorNamespace,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"test-key": "test-value",
				},
			}

			if err := r.Create(ctx, testSecret); err != nil {
				t.Fatalf("Failed to create test Secret: %v", err)
			}

			deletor, err := NewDeletor(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create deletor: %v", err)
			}

			err = deletor.Delete(ctx, DeleteOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Secret",
				},
				Namespace: deletorNamespace,
				Name:      "deletor-test-secret",
			})

			if err != nil {
				t.Fatalf("Failed to delete Secret: %v", err)
			}

			// Wait for deletion to propagate
			time.Sleep(2 * time.Second)

			// Verify the Secret was deleted
			var secret corev1.Secret
			err = r.Get(ctx, "deletor-test-secret", deletorNamespace, &secret)
			if err == nil {
				t.Fatal("Expected Secret to be deleted, but it still exists")
			}

			if !errors.IsNotFound(err) {
				t.Fatalf("Expected NotFound error, got: %v", err)
			}

			t.Logf("Secret deletion test passed")
			return ctx
		}).
		Assess("Delete non-existent resource", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			deletor, err := NewDeletor(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create deletor: %v", err)
			}

			err = deletor.Delete(ctx, DeleteOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ConfigMap",
				},
				Namespace: deletorNamespace,
				Name:      "non-existent-configmap",
			})

			// Deletion of non-existent resource should not return an error
			// (some implementations might return NotFound, others might succeed silently)
			if err != nil && !errors.IsNotFound(err) {
				t.Fatalf("Unexpected error when deleting non-existent resource: %v", err)
			}

			t.Logf("Non-existent resource deletion test passed")
			return ctx
		}).
		Assess("Delete multiple resources", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create multiple test resources
			for i := 1; i <= 3; i++ {
				name := fmt.Sprintf("deletor-test-multi-%d", i)
				err := createTestResource(ctx, cfg, name)
				if err != nil {
					t.Fatalf("Failed to create test ConfigMap %s: %v", name, err)
				}
			}

			deletor, err := NewDeletor(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create deletor: %v", err)
			}

			// Delete all resources
			for i := 1; i <= 3; i++ {
				name := fmt.Sprintf("deletor-test-multi-%d", i)
				err = deletor.Delete(ctx, DeleteOptions{
					GVK: schema.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "ConfigMap",
					},
					Namespace: deletorNamespace,
					Name:      name,
				})

				if err != nil {
					t.Fatalf("Failed to delete ConfigMap %s: %v", name, err)
				}
			}

			// Wait for deletions to propagate
			time.Sleep(3 * time.Second)

			// Verify all resources were deleted
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			for i := 1; i <= 3; i++ {
				name := fmt.Sprintf("deletor-test-multi-%d", i)
				var cm corev1.ConfigMap
				err = r.Get(ctx, name, deletorNamespace, &cm)
				if err == nil {
					t.Fatalf("Expected ConfigMap %s to be deleted, but it still exists", name)
				}

				if !errors.IsNotFound(err) {
					t.Fatalf("Expected NotFound error for %s, got: %v", name, err)
				}
			}

			t.Logf("Multiple resources deletion test passed")
			return ctx
		}).
		Feature()

	deletorTestEnv.Test(t, feature)
}
