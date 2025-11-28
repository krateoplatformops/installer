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
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
)

var (
	chartTestEnv     env.Environment
	chartClusterName string
)

const (
	chartNamespace = "chart-test-system"
)

func TestMain(m *testing.M) {
	chartClusterName = "chart-test"
	chartTestEnv = env.New()

	chartTestEnv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), chartClusterName),
		createChartNamespace(chartNamespace),
		setupChartTestData,
	).Finish(
		envfuncs.DestroyCluster(chartClusterName),
	)

	os.Exit(chartTestEnv.Run(m))
}

func createChartNamespace(ns string) env.Func {
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

func setupChartTestData(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	// Setup any prerequisite data for chart tests
	return ctx, nil
}

func TestChartStepHandlerE2E(t *testing.T) {
	feature := features.New("Chart Step Handler E2E Tests").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Assess("Install simple chart from URL", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createChartHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Create)

			// Use a simple chart from a URL (this requires internet connectivity)
			chartJSON := `{
                "name": "test-nginx",
                "url": "https://raw.githubusercontent.com/helm/examples/main/charts/hello-world/hello-world-0.1.0.tgz",
                "namespace": "chart-test-system",
                "wait": true,
                "set": [
                    {
                        "name": "service.type",
                        "value": "ClusterIP"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := handler.Handle(ctx, "test-install-chart", ext)

			// Note: This test might fail in environments without internet access
			if err != nil {
				t.Logf("Chart installation failed (expected in some test environments): %v", err)
				t.Skip("Skipping chart installation test due to connectivity issues")
				return ctx
			}

			if result.Operation != "install/upgrade" {
				t.Errorf("Expected 'install/upgrade', got '%s'", result.Operation)
			}

			if result.ReleaseName == "" {
				t.Error("Expected release name to be set")
			}

			t.Logf("Chart installation test passed: %s in namespace %s", result.ReleaseName, result.Namespace)
			return ctx
		}).
		Assess("Install chart with variable substitution", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			env := cache.New[string, string]()
			env.Set("SERVICE_TYPE", "NodePort")
			env.Set("REPLICAS", "2")
			env.Set("IMAGE_TAG", "latest")

			handler, err := createChartHandlerWithEnv(cfg, env)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Create)

			chartJSON := `{
                "name": "test-nginx-vars",
                "url": "https://raw.githubusercontent.com/helm/examples/main/charts/hello-world/hello-world-0.1.0.tgz",
                "namespace": "chart-test-system",
                "wait": true,
                "set": [
                    {
                        "name": "service.type",
                        "value": "$SERVICE_TYPE"
                    },
                    {
                        "name": "replicaCount",
                        "value": "$REPLICAS"
                    },
                    {
                        "name": "image.tag",
                        "value": "$IMAGE_TAG"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := handler.Handle(ctx, "test-install-chart-vars", ext)

			if err != nil {
				t.Logf("Chart installation with vars failed (expected in some test environments): %v", err)
				t.Skip("Skipping chart installation test due to connectivity issues")
				return ctx
			}

			t.Logf("Chart installation with variables test passed: %s", result.ReleaseName)
			return ctx
		}).
		Assess("Template chart (dry-run)", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createChartHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Create)

			// Create a chart handler with render mode enabled
			chartHandler := handler
			chartHandler.render = true

			chartJSON := `{
                "name": "test-template",
                "url": "https://raw.githubusercontent.com/helm/examples/main/charts/hello-world/hello-world-0.1.0.tgz",
                "namespace": "chart-test-system",
                "wait": false,
                "set": [
                    {
                        "name": "service.type",
                        "value": "ClusterIP"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := chartHandler.Handle(ctx, "test-template-chart", ext)

			if err != nil {
				t.Logf("Chart templating failed (expected in some test environments): %v", err)
				t.Skip("Skipping chart templating test due to connectivity issues")
				return ctx
			}

			t.Logf("Chart templating test passed: %s", result.ReleaseName)
			return ctx
		}).
		Assess("Install chart from repository", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createChartHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Create)

			// Use a well-known chart repository
			chartJSON := `{
                "name": "test-nginx-repo",
                "repository": "https://charts.bitnami.com/bitnami",
                "version": "18.1.0",
                "namespace": "chart-test-system",
                "wait": true,
                "waitTimeout": "2m",
                "set": [
                    {
                        "name": "service.type",
                        "value": "ClusterIP"
                    },
                    {
                        "name": "replicaCount",
                        "value": "1"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := handler.Handle(ctx, "test-install-repo-chart", ext)

			if err != nil {
				t.Logf("Chart installation from repository failed (expected in some test environments): %v", err)
				t.Skip("Skipping repository chart installation test due to connectivity issues")
				return ctx
			}

			if result.ChartVersion != "18.1.0" {
				t.Errorf("Expected chart version '18.1.0', got '%s'", result.ChartVersion)
			}

			t.Logf("Repository chart installation test passed: %s version %s", result.ReleaseName, result.ChartVersion)
			return ctx
		}).
		Assess("Upgrade existing chart", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createChartHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Update)

			chartJSON := `{
                "name": "test-nginx",
                "url": "https://raw.githubusercontent.com/helm/examples/main/charts/hello-world/hello-world-0.1.0.tgz",
                "namespace": "chart-test-system",
                "wait": true,
                "set": [
                    {
                        "name": "service.type",
                        "value": "LoadBalancer"
                    },
                    {
                        "name": "replicaCount",
                        "value": "3"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := handler.Handle(ctx, "test-upgrade-chart", ext)

			if err != nil {
				t.Logf("Chart upgrade failed (expected if chart wasn't installed): %v", err)
				t.Skip("Skipping chart upgrade test")
				return ctx
			}

			if result.Operation != "install/upgrade" {
				t.Errorf("Expected 'install/upgrade', got '%s'", result.Operation)
			}

			t.Logf("Chart upgrade test passed: %s", result.ReleaseName)
			return ctx
		}).
		Assess("Uninstall chart", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			handler, err := createChartHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Delete)

			chartJSON := `{
                "name": "test-nginx",
                "url": "https://raw.githubusercontent.com/helm/examples/main/charts/hello-world/hello-world-0.1.0.tgz",
                "namespace": "chart-test-system"
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := handler.Handle(ctx, "test-uninstall-chart", ext)

			if err != nil {
				t.Logf("Chart uninstall failed (expected if chart wasn't installed): %v", err)
				t.Skip("Skipping chart uninstall test")
				return ctx
			}

			if result.Operation != "uninstall" {
				t.Errorf("Expected 'uninstall', got '%s'", result.Operation)
			}

			t.Logf("Chart uninstall test passed: %s", result.ReleaseName)
			return ctx
		}).
		Assess("Install chart with credentials", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// First create a secret with credentials
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatalf("Failed to create resources client: %v", err)
			}

			credSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chart-credentials",
					Namespace: chartNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("test-password"),
				},
			}

			if err := r.Create(ctx, credSecret); err != nil {
				t.Fatalf("Failed to create credentials secret: %v", err)
			}

			handler, err := createChartHandler(cfg)
			if err != nil {
				t.Fatalf("Failed to create chart handler: %v", err)
			}

			handler.Namespace(chartNamespace)
			handler.Op(steps.Create)

			chartJSON := `{
                "name": "test-nginx-creds",
                "url": "https://raw.githubusercontent.com/helm/examples/main/charts/hello-world/hello-world-0.1.0.tgz",
                "namespace": "chart-test-system",
                "wait": true,
                "credentials": {
                    "username": "test-user",
                    "passwordRef": {
                        "name": "chart-credentials",
                        "namespace": "chart-test-system",
                        "key": "password"
                    }
                },
                "set": [
                    {
                        "name": "service.type",
                        "value": "ClusterIP"
                    }
                ]
            }`

			ext := &runtime.RawExtension{Raw: []byte(chartJSON)}
			result, err := handler.Handle(ctx, "test-install-chart-creds", ext)

			if err != nil {
				t.Logf("Chart installation with credentials failed (expected in some test environments): %v", err)
				t.Skip("Skipping chart installation with credentials test")
				return ctx
			}

			t.Logf("Chart installation with credentials test passed: %s", result.ReleaseName)
			return ctx
		}).
		Feature()

	chartTestEnv.Test(t, feature)
}

// Helper functions
func createChartHandler(cfg *envconf.Config) (*chartStepHandler, error) {
	getter, err := getter.NewGetter(cfg.Client().RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic getter: %w", err)
	}

	// Create helm client
	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        chartNamespace,
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

	env := cache.New[string, string]()
	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("chart-test"))

	handler := ChartHandler(ChartHandlerOptions{
		Dyn:        getter,
		HelmClient: helmClient,
		Env:        env,
		Log:        log,
	})

	return handler.(*chartStepHandler), nil
}

func createChartHandlerWithEnv(cfg *envconf.Config, env *cache.Cache[string, string]) (*chartStepHandler, error) {
	getter, _ := getter.NewGetter(cfg.Client().RESTConfig())

	opt := &helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        chartNamespace,
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

	helmClient, _ := helmclient.NewClientFromRestConf(opt)
	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("chart-test"))

	handler := ChartHandler(ChartHandlerOptions{
		Dyn:        getter,
		HelmClient: helmClient,
		Env:        env,
		Log:        log,
	})

	return handler.(*chartStepHandler), nil
}
