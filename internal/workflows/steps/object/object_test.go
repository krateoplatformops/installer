//go:build integration
// +build integration

package steps

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"

	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic/applier"
	"github.com/krateoplatformops/installer/internal/dynamic/deletor"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
)

var (
	objTestEnv     env.Environment
	objClusterName string
)

const (
	objNamespace = "object-test-system"
)

func TestMain(m *testing.M) {
	objClusterName = "object-test"
	objTestEnv = env.New()

	objTestEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), objClusterName),
		createObjectNamespace(objNamespace),
		setupObjectTestData,
	).Finish(
		envfuncs.DestroyCluster(objClusterName),
	)

	os.Exit(objTestEnv.Run(m))
}

func createObjectNamespace(ns string) env.Func {
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

func setupObjectTestData(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	// Setup any prerequisite data for object tests
	return ctx, nil
}

func TestObjectStepHandlerE2E(t *testing.T) {
	feature := features.New("Object Step Handler E2E Tests").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Create ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createObjectHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create object handler: %v", err)
			}

			handler.Namespace(objNamespace)
			handler.Op(steps.Create)

			objJSON := `{
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "test-configmap-create",
                    "namespace": "object-test-system"
                },
                "set": [
                    {
                        "name": "data.key1",
                        "value": "value1"
                    },
                    {
                        "name": "data.key2",
                        "value": "value2"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(objJSON)}
			result, err := handler.Handle(ctx, "test-create-cm", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			if result.Operation != "apply" {
				t.Errorf("Expected 'apply', got '%s'", result.Operation)
			}

			if result.Kind != "ConfigMap" {
				t.Errorf("Expected 'ConfigMap', got '%s'", result.Kind)
			}

			// Verify the ConfigMap was actually created
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "test-configmap-create", objNamespace, &cm); err != nil {
				t.Fatalf("Failed to get created ConfigMap: %v", err)
			}

			if cm.Data["key1"] != "value1" {
				t.Errorf("Expected 'value1', got '%s'", cm.Data["key1"])
			}

			t.Logf("ConfigMap creation test passed: %s/%s", result.Namespace, result.Name)
			return ctx
		}).
		Assess("Create Secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createObjectHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create object handler: %v", err)
			}

			handler.Namespace(objNamespace)
			handler.Op(steps.Create)

			objJSON := `{
                "apiVersion": "v1",
                "kind": "Secret",
                "metadata": {
                    "name": "test-secret-create",
                    "namespace": "object-test-system"
                },
                "set": [
                    {
                        "name": "type",
                        "value": "Opaque"
                    },
                    {
                        "name": "stringData.username",
                        "value": "admin"
                    },
                    {
                        "name": "stringData.password",
                        "value": "secret123"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(objJSON)}
			result, err := handler.Handle(ctx, "test-create-secret", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			// Verify the Secret was actually created
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var secret corev1.Secret
			if err := r.Get(ctx, "test-secret-create", objNamespace, &secret); err != nil {
				t.Fatalf("Failed to get created Secret: %v", err)
			}

			if string(secret.Data["username"]) != "admin" {
				t.Errorf("Expected 'admin', got '%s'", string(secret.Data["username"]))
			}

			t.Logf("Secret creation test passed: %s/%s", result.Namespace, result.Name)
			return ctx
		}).
		Assess("Update ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createObjectHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create object handler: %v", err)
			}

			handler.Namespace(objNamespace)
			handler.Op(steps.Update)

			objJSON := `{
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "test-configmap-create",
                    "namespace": "object-test-system"
                },
                "set": [
                    {
                        "name": "data.key1",
                        "value": "updated-value1"
                    },
                    {
                        "name": "data.key3",
                        "value": "new-value3"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(objJSON)}
			result, err := handler.Handle(ctx, "test-update-cm", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			// Verify the ConfigMap was updated
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			// Wait a bit for the update to propagate
			time.Sleep(1 * time.Second)

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "test-configmap-create", objNamespace, &cm); err != nil {
				t.Fatalf("Failed to get updated ConfigMap: %v", err)
			}

			if cm.Data["key1"] != "updated-value1" {
				t.Errorf("Expected 'updated-value1', got '%s'", cm.Data["key1"])
			}

			if cm.Data["key3"] != "new-value3" {
				t.Errorf("Expected 'new-value3', got '%s'", cm.Data["key3"])
			}

			t.Logf("ConfigMap update test passed: %s/%s", result.Namespace, result.Name)
			return ctx
		}).
		Assess("Create with variable substitution", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			env := cache.New[string, string]()
			env.Set("APP_NAME", "my-application")
			env.Set("VERSION", "v1.2.3")
			env.Set("REPLICAS", "3")

			handler, err := createObjectHandlerWithEnv(cfg, env)
			if err != nil {
				t.Fatalf("Failed to create object handler: %v", err)
			}

			handler.Namespace(objNamespace)
			handler.Op(steps.Create)

			objJSON := `{
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "test-configmap-vars",
                    "namespace": "object-test-system"
                },
                "set": [
                    {
                        "name": "data.app-name",
                        "value": "$APP_NAME"
                    },
                    {
                        "name": "data.version",
                        "value": "$VERSION"
                    },
                    {
                        "name": "data.replicas",
                        "value": "$REPLICAS",
						"asString": true
                    },
                    {
                        "name": "data.full-image",
                        "value": "registry.io/$APP_NAME:$VERSION"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(objJSON)}
			result, err := handler.Handle(ctx, "test-vars-cm", ext)
			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			// Verify the ConfigMap with substituted values
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "test-configmap-vars", objNamespace, &cm); err != nil {
				t.Fatalf("Failed to get ConfigMap with vars: %v", err)
			}

			if cm.Data["app-name"] != "my-application" {
				t.Errorf("Expected 'my-application', got '%s'", cm.Data["app-name"])
			}

			if cm.Data["full-image"] != "registry.io/my-application:v1.2.3" {
				t.Errorf("Expected 'registry.io/my-application:v1.2.3', got '%s'", cm.Data["full-image"])
			}

			t.Logf("Variable substitution test passed: %s/%s", result.Namespace, result.Name)
			return ctx
		}).
		Assess("Delete ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createObjectHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create object handler: %v", err)
			}

			handler.Namespace(objNamespace)
			handler.Op(steps.Delete)

			objJSON := `{
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "test-configmap-vars",
                    "namespace": "object-test-system"
                },
                "set": []
            }`

			ext := &runtime.RawExtension{Raw: []byte(objJSON)}
			result, err := handler.Handle(ctx, "test-delete-cm", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			if result.Operation != "delete" {
				t.Errorf("Expected 'delete', got '%s'", result.Operation)
			}

			time.Sleep(5 * time.Second) // Wait for deletion to propagate

			// Verify the ConfigMap was deleted
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "test-configmap-vars", objNamespace, &cm); err == nil {
				t.Error("ConfigMap should have been deleted but still exists")
			}

			t.Logf("ConfigMap deletion test passed: %s/%s", result.Namespace, result.Name)
			return ctx
		}).
		Assess("Create with default namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createObjectHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create object handler: %v", err)
			}

			handler.Namespace(objNamespace)
			handler.Op(steps.Create)

			// No namespace specified in metadata, should use handler's default
			objJSON := `{
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": "test-configmap-default-ns"
                },
                "set": [
                    {
                        "name": "data.namespace-test",
                        "value": "default-namespace-value"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(objJSON)}
			result, err := handler.Handle(ctx, "test-default-ns-cm", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			if result.Namespace != objNamespace {
				t.Errorf("Expected namespace '%s', got '%s'", objNamespace, result.Namespace)
			}

			// Verify the ConfigMap was created in the correct namespace
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "test-configmap-default-ns", objNamespace, &cm); err != nil {
				t.Fatalf("Failed to get ConfigMap in default namespace: %v", err)
			}

			t.Logf("Default namespace test passed: %s/%s", result.Namespace, result.Name)
			return ctx
		}).
		Feature()

	objTestEnv.Test(t, feature)
}

// Helper functions
func createObjectHandler(cfg *envconf.Config) (*objStepHandler, error) {
	applier, err := applier.NewApplier(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic applier: %w", err)
	}

	deletor, err := deletor.NewDeletor(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic deletor: %w", err)
	}

	env := cache.New[string, string]()
	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("object-test"))

	handler := ObjectHandler(applier, deletor, env, log)
	return handler.(*objStepHandler), nil
}

func createObjectHandlerWithEnv(cfg *envconf.Config, env *cache.Cache[string, string]) (*objStepHandler, error) {
	applier, _ := applier.NewApplier(cfg.Client().RESTConfig())
	deletor, _ := deletor.NewDeletor(cfg.Client().RESTConfig())
	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("object-test"))

	handler := ObjectHandler(applier, deletor, env, log)
	return handler.(*objStepHandler), nil
}
