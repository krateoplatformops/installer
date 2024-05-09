//go:build integration
// +build integration

package steps

import (
	"fmt"
	"os"
	"testing"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/ptr"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"helm.sh/helm/v3/pkg/action"
	pkgcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
)

func TestChartTemplate(t *testing.T) {
	const (
		ns        = "demo-system"
		chartPath = "../../../testdata/chart-unit-test"
	)

	data := []*v1alpha1.Data{
		{
			Name:     "sftp.allowedMACs",
			Value:    "null",
			AsString: ptr.To(true),
		},
		{
			Name:     "securityContext.runAsUser",
			Value:    "null",
			AsString: ptr.To(true),
		},
		{
			Name:  "env.KRATEO_GATEWAY_DNS_NAMES",
			Value: "{krateo-gateway.sticz.svc,$KRATEO_GATEWAY_INGRESS_HOST}",
		},
	}

	env := cache.New[string, string]()
	env.Set("KRATEO_GATEWAY_INGRESS_HOST", "http://sti.cz")

	hdl, err := chartStepHandlerForNamespace(ns, env)
	if err != nil {
		t.Fatal(err)
	}

	opts := hdl.valuesOptions("test", data)

	yml, err := renderChart(&helmclient.ChartSpec{
		Namespace:     ns,
		ValuesOptions: opts,
	}, ns, chartPath)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(yml)
}

func chartStepHandlerForNamespace(ns string, env *cache.Cache[string, string]) (*chartStepHandler, error) {
	rc, err := newRestConfig()
	if err != nil {
		return nil, err
	}

	cli, err := helmClientForNamespace(rc, ns)
	if err != nil {
		return nil, err
	}

	logr := logging.NewLogrLogger(newStdoutLogger())

	return &chartStepHandler{
		cli: cli, env: env,
		subst: func(k string) string {
			if v, ok := env.Get(k); ok {
				return v
			}

			return "$" + k
		},
		logr: logr,
	}, nil
}

func renderChart(spec *helmclient.ChartSpec, ns, chartPath string) (string, error) {
	p := getter.All(pkgcli.New())
	values, err := spec.GetValuesMap(p)
	if err != nil {
		return "", err
	}

	chart, err := loadChart(os.DirFS(chartPath))
	if err != nil {
		return "", err
	}

	client := action.NewInstall(&action.Configuration{})
	client.ClientOnly = true
	client.DryRun = true
	client.ReleaseName = "test"
	client.IncludeCRDs = true
	client.Namespace = ns

	rel, err := client.Run(chart, values)
	if err != nil {
		return "", fmt.Errorf("could not render helm chart correctly: %w", err)
	}
	return rel.Manifest, nil
}
