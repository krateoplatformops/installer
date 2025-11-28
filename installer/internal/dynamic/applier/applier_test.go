//go:build integration
// +build integration

package applier

import (
	"context"
	"os"
	"testing"
	"time"

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
	applierTestEnv     env.Environment
	applierClusterName string
)

const (
	applierNamespace = "applier-test-system"
)

func TestMain(m *testing.M) {
	applierClusterName = "applier-test"
	applierTestEnv = env.New()

	applierTestEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), applierClusterName),
		createApplierNamespace(applierNamespace),
	).Finish(
		envfuncs.DestroyCluster(applierClusterName),
	)

	os.Exit(applierTestEnv.Run(m))
}

func createApplierNamespace(ns string) env.Func {
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

func TestApplierE2E(t *testing.T) {
	feature := features.New("Applier E2E Tests").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Apply ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			applier, err := NewApplier(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create applier: %v", err)
			}

			content := map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "applier-test-cm",
					"namespace": applierNamespace,
				},
				"data": map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
			}

			err = applier.Apply(ctx, content, ApplyOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ConfigMap",
				},
				Namespace: applierNamespace,
				Name:      "applier-test-cm",
			})

			if err != nil {
				t.Fatalf("Failed to apply ConfigMap: %v", err)
			}

			// Verify the ConfigMap was created
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "applier-test-cm", applierNamespace, &cm); err != nil {
				t.Fatalf("Failed to get ConfigMap: %v", err)
			}

			if cm.Data["key1"] != "value1" {
				t.Errorf("Expected 'value1', got '%s'", cm.Data["key1"])
			}

			t.Logf("ConfigMap apply test passed: %s", cm.Name)
			return ctx
		}).
		Assess("Apply Secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			applier, err := NewApplier(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create applier: %v", err)
			}

			content := map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "applier-test-secret",
					"namespace": applierNamespace,
				},
				"type": "Opaque",
				"stringData": map[string]any{
					"username": "admin",
					"password": "secret123",
				},
			}

			err = applier.Apply(ctx, content, ApplyOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Secret",
				},
				Namespace: applierNamespace,
				Name:      "applier-test-secret",
			})

			if err != nil {
				t.Fatalf("Failed to apply Secret: %v", err)
			}

			// Verify the Secret was created
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var secret corev1.Secret
			if err := r.Get(ctx, "applier-test-secret", applierNamespace, &secret); err != nil {
				t.Fatalf("Failed to get Secret: %v", err)
			}

			if string(secret.Data["username"]) != "admin" {
				t.Errorf("Expected 'admin', got '%s'", string(secret.Data["username"]))
			}

			t.Logf("Secret apply test passed: %s", secret.Name)
			return ctx
		}).
		Assess("Update existing resource", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			applier, err := NewApplier(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create applier: %v", err)
			}

			// Update the ConfigMap with new data
			content := map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "applier-test-cm",
					"namespace": applierNamespace,
				},
				"data": map[string]any{
					"key1":    "updated-value1",
					"key2":    "value2",
					"new-key": "new-value",
				},
			}

			err = applier.Apply(ctx, content, ApplyOptions{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ConfigMap",
				},
				Namespace: applierNamespace,
				Name:      "applier-test-cm",
			})

			if err != nil {
				t.Fatalf("Failed to update ConfigMap: %v", err)
			}

			// Wait for update to propagate
			time.Sleep(1 * time.Second)

			// Verify the ConfigMap was updated
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "applier-test-cm", applierNamespace, &cm); err != nil {
				t.Fatalf("Failed to get updated ConfigMap: %v", err)
			}

			if cm.Data["key1"] != "updated-value1" {
				t.Errorf("Expected 'updated-value1', got '%s'", cm.Data["key1"])
			}

			if cm.Data["new-key"] != "new-value" {
				t.Errorf("Expected 'new-value', got '%s'", cm.Data["new-key"])
			}

			t.Logf("ConfigMap update test passed: %s", cm.Name)
			return ctx
		}).
		Feature()

	applierTestEnv.Test(t, feature)
}
