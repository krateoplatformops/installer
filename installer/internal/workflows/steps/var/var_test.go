//go:build integration
// +build integration

package steps

import (
	"context"
	"fmt"
	"os"
	"testing"

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
	"github.com/krateoplatformops/installer/internal/dynamic/getter"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
)

var (
	varTestEnv     env.Environment
	varClusterName string
)

const (
	varNamespace = "var-test-system"
)

func TestMain(m *testing.M) {
	varClusterName = "var-test"
	varTestEnv = env.New()

	varTestEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), varClusterName),
		createVarNamespace(varNamespace),
		setupVarTestData,
	).Finish(
		envfuncs.DestroyCluster(varClusterName),
	)

	os.Exit(varTestEnv.Run(m))
}

func createVarNamespace(ns string) env.Func {
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

func setupVarTestData(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return ctx, err
	}

	// Create test ConfigMap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "var-test-config",
			Namespace: varNamespace,
		},
		Data: map[string]string{
			"database-url":  "postgres://localhost:5432/mydb",
			"service-port":  "8080",
			"feature-flag":  "true",
			"nested.config": `{"key": "extracted-nested-value"}`,
			"api-key":       "secret-api-key-123",
		},
	}

	if err := r.Create(ctx, testConfigMap); err != nil {
		return ctx, fmt.Errorf("failed to create var test ConfigMap: %w", err)
	}

	// Create test Secret
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "var-test-secret",
			Namespace: varNamespace,
		},
		Data: map[string][]byte{
			"password":    []byte("super-secret-password"),
			"private-key": []byte("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKB\n-----END PRIVATE KEY-----"),
			"token":       []byte("bearer-token-xyz789"),
		},
	}

	if err := r.Create(ctx, testSecret); err != nil {
		return ctx, fmt.Errorf("failed to create var test Secret: %w", err)
	}

	return ctx, nil
}

func TestVarStepHandlerE2E(t *testing.T) {
	feature := features.New("Var Step Handler E2E Tests").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Variable with direct value", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			varJSON := `{
                "name": "DIRECT_VALUE",
                "value": "simple-direct-value"
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			result, err := handler.Handle(ctx, "test-direct-value", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			if result.Name != "DIRECT_VALUE" {
				t.Errorf("Expected 'DIRECT_VALUE', got '%s'", result.Name)
			}

			if result.Value != "simple-direct-value" {
				t.Errorf("Expected 'simple-direct-value', got '%s'", result.Value)
			}

			t.Logf("Direct value test passed: %s = %s", result.Name, result.Value)
			return ctx
		}).
		Assess("Variable with substitution", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			// Pre-populate environment
			env := cache.New[string, string]()
			env.Set("PROTOCOL", "https")
			env.Set("HOST", "api.example.com")
			env.Set("PORT", "443")

			handler = createVarHandlerWithEnv(cfg, env)
			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			varJSON := `{
                "name": "FULL_URL",
                "value": "$PROTOCOL://$HOST:$PORT/v1/api"
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			result, err := handler.Handle(ctx, "test-substitution", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			expected := "https://api.example.com:443/v1/api"
			if result.Value != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result.Value)
			}

			t.Logf("Substitution test passed: %s = %s", result.Name, result.Value)
			return ctx
		}).
		Assess("Variable from ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			varJSON := `{
                "name": "DATABASE_URL",
                "valueFrom": {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "var-test-config",
                        "namespace": "var-test-system"
                    },
                    "selector": ".data.\"database-url\""
                }
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			result, err := handler.Handle(ctx, "test-configmap", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			expected := "postgres://localhost:5432/mydb"
			if result.Value != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result.Value)
			}

			t.Logf("ConfigMap extraction test passed: %s = %s", result.Name, result.Value)
			return ctx
		}).
		Assess("Variable from Secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			varJSON := `{
                "name": "SECRET_PASSWORD",
                "valueFrom": {
                    "apiVersion": "v1",
                    "kind": "Secret",
                    "metadata": {
                        "name": "var-test-secret",
                        "namespace": "var-test-system"
                    },
                    "selector": ".data.password | @base64d"
                }
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			result, err := handler.Handle(ctx, "test-secret", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			// Secret data is base64 encoded
			expected := "super-secret-password"
			if result.Value != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result.Value)
			}

			t.Logf("Secret extraction test passed: %s = %s", result.Name, result.Value)
			return ctx
		}).
		Assess("Variable from non-existent resource", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			varJSON := `{
                "name": "MISSING_VALUE",
                "valueFrom": {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "non-existent-config",
                        "namespace": "var-test-system"
                    },
                    "selector": ".data.\"missing-key\""
                }
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			_, err = handler.Handle(ctx, "test-missing", ext)

			if err == nil {
				t.Fatal("Expected error for non-existent resource, got nil")
			}

			t.Logf("Non-existent resource test passed: got expected error: %v", err)
			return ctx
		}).
		Assess("Variable with complex JSON selector", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			varJSON := `{
                "name": "NESTED_VALUE",
                "valueFrom": {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "var-test-config",
                        "namespace": "var-test-system"
                    },
                    "selector": ".data.\"nested.config\" | fromjson | .key"
                }
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			result, err := handler.Handle(ctx, "test-json-selector", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			expected := "extracted-nested-value"
			if result.Value != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result.Value)
			}

			t.Logf("JSON selector test passed: %s = %s", result.Name, result.Value)
			return ctx
		}).
		Assess("Variable with default namespace", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createVarHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create var handler: %v", err)
			}

			handler.Namespace(varNamespace)
			handler.Op(steps.Create)

			// No namespace specified, should use handler's default namespace
			varJSON := `{
                "name": "DEFAULT_NS_VALUE",
                "valueFrom": {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "var-test-config"
                    },
                    "selector": ".data.\"service-port\""
                }
            }`

			ext := &runtime.RawExtension{Raw: []byte(varJSON)}
			result, err := handler.Handle(ctx, "test-default-ns", ext)

			if err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			expected := "8080"
			if result.Value != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result.Value)
			}

			t.Logf("Default namespace test passed: %s = %s", result.Name, result.Value)
			return ctx
		}).
		Feature()

	varTestEnv.Test(t, feature)
}

// Helper functions
func createVarHandler(cfg *envconf.Config) (*varStepHandler, error) {
	getter, err := getter.NewGetter(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic getter: %w", err)
	}

	env := cache.New[string, string]()
	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("var-test"))

	handler := VarHandler(getter, env, log)
	return handler.(*varStepHandler), nil
}

func createVarHandlerWithEnv(cfg *envconf.Config, env *cache.Cache[string, string]) *varStepHandler {
	getter, _ := getter.NewGetter(cfg.Client().RESTConfig())
	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("var-test"))

	handler := VarHandler(getter, env, log)
	return handler.(*varStepHandler)
}
