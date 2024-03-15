package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/helmclient/values"
	"github.com/krateoplatformops/installer/internal/ptr"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime"
)

func ChartHandler(cli helmclient.Client, env *cache.Cache[string, string]) Handler {
	return &chartStepHandler{
		cli: cli, env: env,
	}
}

var _ Handler = (*chartStepHandler)(nil)

type chartStepHandler struct {
	cli helmclient.Client
	env *cache.Cache[string, string]
	ns  string
}

func (r *chartStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *chartStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) error {
	res := v1alpha1.ChartSpec{}
	err := json.Unmarshal(ext.Raw, &res)
	if err != nil {
		return err
	}

	entry := repo.Entry{
		Name: deriveRepoName(res.Repository),
		URL:  res.Repository,
	}
	err = r.cli.AddOrUpdateChartRepo(entry)
	if err != nil {
		return err
	}

	timeout := time.Duration(10 * time.Minute)
	if res.WaitTimeout != nil {
		timeout = res.WaitTimeout.Duration
	}

	spec := helmclient.ChartSpec{
		ReleaseName:     res.Name,
		ChartName:       fmt.Sprintf("%s/%s", entry.Name, res.Name),
		Namespace:       r.ns,
		Version:         res.Version,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		Wait:            ptr.Deref(res.Wait, true),
		ValuesOptions: values.Options{
			Values: resolveVars(res.Set, r.env),
		},
		Timeout: timeout,
	}

	_, err = r.cli.InstallOrUpgradeChart(ctx, &spec, nil)
	return err
}
