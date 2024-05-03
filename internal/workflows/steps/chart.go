package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/expand"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/helmclient/values"
	"github.com/krateoplatformops/installer/internal/ptr"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime"
)

func ChartHandler(cli helmclient.Client, env *cache.Cache[string, string], verbose bool) Handler {
	return &chartStepHandler{
		cli: cli, env: env,
		subst: func(k string) string {
			if v, ok := env.Get(k); ok {
				return v
			}

			return "$" + k
		},
		verbose: verbose,
	}
}

var _ Handler = (*chartStepHandler)(nil)

type chartStepHandler struct {
	cli     helmclient.Client
	env     *cache.Cache[string, string]
	ns      string
	op      Op
	subst   func(k string) string
	verbose bool
}

func (r *chartStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *chartStepHandler) Op(op Op) {
	r.op = op
}

func (r *chartStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) error {
	spec, err := r.toChartSpec(id, ext)
	if err != nil {
		return err
	}

	if r.op != Delete {
		_, err = r.cli.InstallOrUpgradeChart(ctx, spec, nil)
		return err
	}

	err = r.cli.UninstallRelease(spec)
	if err != nil {
		log.Printf("WARN: %s (%s)", err.Error(), spec.ChartName)
		if strings.Contains(err.Error(), "release: not found") {
			return nil
		}
		return err
	}

	return nil
}

func (r *chartStepHandler) toChartSpec(id string, ext *runtime.RawExtension) (*helmclient.ChartSpec, error) {
	res := v1alpha1.ChartSpec{}
	err := json.Unmarshal(ext.Raw, &res)
	if err != nil {
		return nil, err
	}

	entry := repo.Entry{
		Name: deriveRepoName(res.Repository),
		URL:  res.Repository,
	}

	if r.op != Delete {
		err = r.cli.AddOrUpdateChartRepo(entry)
		if err != nil {
			return nil, err
		}
	}

	timeout := time.Duration(10 * time.Minute)
	if res.WaitTimeout != nil {
		timeout = res.WaitTimeout.Duration
	}

	spec := &helmclient.ChartSpec{
		ReleaseName:     res.Name,
		ChartName:       fmt.Sprintf("%s/%s", entry.Name, res.Name),
		Namespace:       r.ns,
		Version:         res.Version,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		Wait:            ptr.Deref(res.Wait, true),
		ValuesOptions:   r.valuesOptions(id, res.Set),
		Timeout:         timeout,
	}

	return spec, nil
}

func (r *chartStepHandler) valuesOptions(id string, res []*v1alpha1.Data) (opts values.Options) {
	opts.StringValues = []string{}
	opts.Values = []string{}

	for _, el := range res {
		if len(el.Value) > 0 {
			val := expand.Expand(el.Value, "", r.subst)
			line := fmt.Sprintf("%s=%s", el.Name, val)
			if ptr.Deref(el.AsString, false) {
				opts.StringValues = append(opts.StringValues, line)
			} else {
				opts.Values = append(opts.Values, line)
			}

			if r.verbose {
				log.Printf("DBG [chart:%s]: set (name: %s, value: %s)",
					id, el.Name, ellipsis(strval(val), 20))
			}
		}
	}

	return opts
}
