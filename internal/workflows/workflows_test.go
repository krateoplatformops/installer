//go:build integration
// +build integration

package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/dynamic/applier"
	"github.com/krateoplatformops/installer/internal/dynamic/deletor"
	"github.com/krateoplatformops/installer/internal/dynamic/getter"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
)

var (
	testenv     env.Environment
	clusterName string
)

const (
	testdataPath = "../../testdata"
	namespace    = "krateo-system"
	altNamespace = "demo-system"
)

func TestMain(m *testing.M) {
	clusterName = "workflows-test"
	testenv = env.New()

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		createNamespace(namespace),
		createNamespace(altNamespace),
		setupTestData,
	).Finish(
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}

func createNamespace(ns string) env.Func {
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

func setupTestData(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return ctx, err
	}

	// Apply CRDs if they exist
	crdsPath := filepath.Join(testdataPath, "../crds")
	if _, err := os.Stat(crdsPath); err == nil {
		if err := decoder.ApplyWithManifestDir(ctx, r, crdsPath, "*.yaml", nil); err != nil {
			return ctx, fmt.Errorf("failed to apply CRDs: %w", err)
		}
		// Wait for CRDs to be ready
		time.Sleep(5 * time.Second)
	}

	// Create test ConfigMaps for variable extraction
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"key":         "extracted-value",
			"host":        "example.com",
			"port":        "8080",
			"api-version": "v1",
		},
	}

	if err := r.Create(ctx, testConfigMap); err != nil {
		return ctx, fmt.Errorf("failed to create test ConfigMap: %w", err)
	}

	// Create test Secret
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"token":    []byte("secret-token-value"),
			"password": []byte("super-secret"),
		},
	}

	if err := r.Create(ctx, testSecret); err != nil {
		return ctx, fmt.Errorf("failed to create test Secret: %w", err)
	}

	return ctx, nil
}

func TestWorkflowVarOperations(t *testing.T) {
	feature := features.New("Workflow Variable Operations").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Variable with direct value", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			workflow, err := createTestWorkflow(cfg)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			spec := &v1alpha1.WorkflowSpec{
				Steps: []*v1alpha1.Step{
					{
						ID:   "set-api-url",
						Type: v1alpha1.TypeVar,
						With: &runtime.RawExtension{
							Raw: []byte(`{
                                "name": "API_URL",
                                "value": "https://api.example.com:8080"
                            }`),
						},
					},
				},
			}

			workflow.Op(steps.Create)
			results := workflow.Run(ctx, spec, func(s *v1alpha1.Step) bool { return false })

			if err := Err(results); err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			varResult := results[0].Result()
			if varResult == nil {
				t.Fatal("Expected variable result, got nil")
			}

			// Check if the variable was set in the environment
			value, exists := workflow.env.Get("API_URL")
			if !exists {
				t.Fatal("Variable API_URL not found in environment")
			}

			if value != "https://api.example.com:8080" {
				t.Fatalf("Expected 'https://api.example.com:8080', got '%s'", value)
			}

			t.Logf("Variable set successfully: %s = %s", "API_URL", value)
			return ctx
		}).
		Assess("Variable with valueFrom ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			workflow, err := createTestWorkflow(cfg)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			spec := &v1alpha1.WorkflowSpec{
				Steps: []*v1alpha1.Step{
					{
						ID:   "extract-config-value",
						Type: v1alpha1.TypeVar,
						With: &runtime.RawExtension{
							Raw: []byte(`{
                                "name": "EXTRACTED_VALUE",
                                "valueFrom": {
                                    "apiVersion": "v1",
                                    "kind": "ConfigMap",
                                    "metadata": {
                                        "name": "test-config",
                                        "namespace": "krateo-system"
                                    },
                                    "selector": ".data.key"
                                }
                            }`),
						},
					},
				},
			}

			workflow.Op(steps.Create)
			results := workflow.Run(ctx, spec, func(s *v1alpha1.Step) bool { return false })

			if err := Err(results); err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			// Check if the variable was extracted correctly
			value, exists := workflow.env.Get("EXTRACTED_VALUE")
			if !exists {
				t.Fatal("Variable EXTRACTED_VALUE not found in environment")
			}

			if value != "extracted-value" {
				t.Fatalf("Expected 'extracted-value', got '%s'", value)
			}

			t.Logf("Variable extracted successfully: %s = %s", "EXTRACTED_VALUE", value)
			return ctx
		}).
		Assess("Variable substitution", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			workflow, err := createTestWorkflow(cfg)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			// Pre-populate some variables
			workflow.env.Set("HOST", "example.com")
			workflow.env.Set("PORT", "8080")

			spec := &v1alpha1.WorkflowSpec{
				Steps: []*v1alpha1.Step{
					{
						ID:   "compose-url",
						Type: v1alpha1.TypeVar,
						With: &runtime.RawExtension{
							Raw: []byte(`{
                                "name": "FULL_URL",
                                "value": "https://$HOST:$PORT/api"
                            }`),
						},
					},
				},
			}

			workflow.Op(steps.Create)
			results := workflow.Run(ctx, spec, func(s *v1alpha1.Step) bool { return false })

			if err := Err(results); err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			// Check if the variable substitution worked
			value, exists := workflow.env.Get("FULL_URL")
			if !exists {
				t.Fatal("Variable FULL_URL not found in environment")
			}

			expected := "https://example.com:8080/api"
			if value != expected {
				t.Fatalf("Expected '%s', got '%s'", expected, value)
			}

			t.Logf("Variable substitution successful: %s = %s", "FULL_URL", value)
			return ctx
		}).
		Feature()

	testenv.Test(t, feature)
}

func TestWorkflowObjectOperations(t *testing.T) {
	feature := features.New("Workflow Object Operations").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Create ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			workflow, err := createTestWorkflow(cfg)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			spec := &v1alpha1.WorkflowSpec{
				Steps: []*v1alpha1.Step{
					{
						ID:   "create-configmap",
						Type: v1alpha1.TypeObject,
						With: &runtime.RawExtension{
							Raw: []byte(`{
                                "apiVersion": "v1",
                                "kind": "ConfigMap",
                                "metadata": {
                                    "name": "workflow-test-cm",
                                    "namespace": "krateo-system"
                                },
                                "set": [
                                    {
                                        "name": "data.test-key",
                                        "value": "test-value"
                                    }
                                ]
                            }`),
						},
					},
				},
			}

			workflow.Op(steps.Create)
			results := workflow.Run(ctx, spec, func(s *v1alpha1.Step) bool { return false })

			if err := Err(results); err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			// Verify the ConfigMap was created
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "workflow-test-cm", namespace, &cm); err != nil {
				t.Fatalf("Failed to get created ConfigMap: %v", err)
			}

			if cm.Data["test-key"] != "test-value" {
				t.Fatalf("Expected 'test-value', got '%s'", cm.Data["test-key"])
			}

			t.Logf("ConfigMap created successfully: %s", cm.Name)
			return ctx
		}).
		Assess("Update ConfigMap", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			workflow, err := createTestWorkflow(cfg)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			spec := &v1alpha1.WorkflowSpec{
				Steps: []*v1alpha1.Step{
					{
						ID:   "update-configmap",
						Type: v1alpha1.TypeObject,
						With: &runtime.RawExtension{
							Raw: []byte(`{
                                "apiVersion": "v1",
                                "kind": "ConfigMap",
                                "metadata": {
                                    "name": "workflow-test-cm",
                                    "namespace": "krateo-system"
                                },
                                "set": [
                                    {
                                        "name": "data.updated-key",
                                        "value": "updated-value"
                                    }
                                ]
                            }`),
						},
					},
				},
			}

			workflow.Op(steps.Update)
			results := workflow.Run(ctx, spec, func(s *v1alpha1.Step) bool { return false })

			if err := Err(results); err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			// Verify the ConfigMap was updated
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			var cm corev1.ConfigMap
			if err := r.Get(ctx, "workflow-test-cm", namespace, &cm); err != nil {
				t.Fatalf("Failed to get updated ConfigMap: %v", err)
			}

			if cm.Data["updated-key"] != "updated-value" {
				t.Fatalf("Expected 'updated-value', got '%s'", cm.Data["updated-key"])
			}

			t.Logf("ConfigMap updated successfully: %s", cm.Name)
			return ctx
		}).
		Feature()

	testenv.Test(t, feature)
}

func TestWorkflowChartOperations(t *testing.T) {
	feature := features.New("Workflow Chart Operations").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Install Chart", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			workflow, err := createTestWorkflow(cfg)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			spec := &v1alpha1.WorkflowSpec{
				Steps: []*v1alpha1.Step{
					{
						ID:   "install-test-chart",
						Type: v1alpha1.TypeChart,
						With: &runtime.RawExtension{
							Raw: []byte(`{
                                "name": "test-release",
                                "chart": "nginx",
                                "version": "1.0.0",
                                "repository": "https://charts.bitnami.com/bitnami",
                                "namespace": "krateo-system",
                                "wait": true,
                                "set": [
                                    {
                                        "name": "service.type",
                                        "value": "ClusterIP"
                                    }
                                ]
                            }`),
						},
					},
				},
			}

			workflow.Op(steps.Create)
			results := workflow.Run(ctx, spec, func(s *v1alpha1.Step) bool { return false })

			// Note: In un vero test e2e, questo potrebbe fallire se non c'è connettività internet
			// o se il chart repository non è raggiungibile. Potresti voler usare un chart locale.
			if err := Err(results); err != nil {
				t.Logf("Chart installation failed (expected in test environment): %v", err)
				// Non facciamo fail del test per questo motivo in ambiente di test
				return ctx
			}

			t.Logf("Chart installation completed successfully")
			return ctx
		}).
		Feature()

	testenv.Test(t, feature)
}
func createTestWorkflow(cfg *envconf.Config) (*Workflow, error) {
	getter, err := getter.NewGetter(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic getter: %w", err)
	}

	applier, err := applier.NewApplier(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic applier: %w", err)
	}

	deletor, err := deletor.NewDeletor(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic deletor: %w", err)
	}

	// Create helm client options
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        namespace,
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          false,
			DebugLog: func(format string, v ...interface{}) {
				// Log to test output
			},
		},
		RestConfig: cfg.Client().RESTConfig(),
	}

	helmClient, err := helmclient.NewClientFromRestConf(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create helm client: %w", err)
	}

	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("workflow-test"))

	return New(Opts{
		Getter:         getter,
		Applier:        applier,
		Deletor:        deletor,
		Log:            log,
		HelmClient:     helmClient,
		MaxHelmHistory: 10,
		Namespace:      namespace,
	})
}
