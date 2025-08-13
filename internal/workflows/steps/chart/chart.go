package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/krateoplatformops/installer/apis/workflows/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
	"github.com/krateoplatformops/installer/internal/dynamic/getter"
	"github.com/krateoplatformops/installer/internal/expand"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/helmclient/values"
	"github.com/krateoplatformops/installer/internal/resolvers"
	"github.com/krateoplatformops/installer/internal/workflows/steps"
	"github.com/krateoplatformops/plumbing/ptr"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ChartHandlerOptions struct {
	Dyn        *getter.Getter
	HelmClient helmclient.Client
	Env        *cache.Cache[string, string]
	Log        logging.Logger
}

func ChartHandler(opts ChartHandlerOptions) steps.Handler[*steps.ChartResult] {
	hdl := &chartStepHandler{
		cli:  opts.HelmClient,
		env:  opts.Env,
		logr: opts.Log,
		dyn:  opts.Dyn,
	}
	hdl.subst = func(k string) string {
		if v, ok := hdl.env.Get(k); ok {
			return v
		}

		return "$" + k
	}

	return hdl
}

var _ steps.Handler[*steps.ChartResult] = (*chartStepHandler)(nil)

type chartStepHandler struct {
	cli    helmclient.Client
	env    *cache.Cache[string, string]
	ns     string
	op     steps.Op
	subst  func(k string) string
	render bool
	logr   logging.Logger
	dyn    *getter.Getter
}

func (r *chartStepHandler) Namespace(ns string) {
	r.ns = ns
}

func (r *chartStepHandler) Op(op steps.Op) {
	r.op = op
}

func (r *chartStepHandler) Handle(ctx context.Context, id string, ext *runtime.RawExtension) (*steps.ChartResult, error) {
	spec, err := r.toChartSpec(ctx, id, ext)
	if err != nil {
		return nil, err
	}

	result := &steps.ChartResult{}

	if r.op != steps.Delete {
		result.Operation = "install/upgrade"

		release, err := r.cli.InstallOrUpgradeChart(ctx, spec, nil)
		if err != nil {
			return result, err
		}

		if release != nil {
			result.Status = string(release.Info.Status)
			result.ChartVersion = release.Chart.Metadata.Version
			result.AppVersion = release.Chart.Metadata.AppVersion
			result.ReleaseName = release.Name
			result.ChartName = release.Chart.Metadata.Name
			result.Namespace = release.Namespace
			result.Updated = metav1.NewTime(release.Info.LastDeployed.Time)
			result.Revision = release.Version
		}

		r.logr.Debug(fmt.Sprintf(
			"[chart:%s]: %s operation completed for release %s",
			id, result.Operation, result.ReleaseName))

		return result, nil
	}

	result.Operation = "uninstall"

	err = r.cli.UninstallRelease(spec)
	if err != nil {
		r.logr.Info(fmt.Sprintf("WARN: %s (%s)", err.Error(), spec.ChartName))
		if strings.Contains(err.Error(), "release: not found") {
			result.Status = "not_found"
			return result, nil
		}
		return result, err
	}

	result.Status = "uninstalled"

	r.logr.Debug(fmt.Sprintf(
		"[chart:%s]: uninstall operation completed for release %s",
		id, result.ReleaseName))

	return result, nil
}

func (r *chartStepHandler) toChartSpec(ctx context.Context, id string, ext *runtime.RawExtension) (*helmclient.ChartSpec, error) {
	res := v1alpha1.ChartSpec{}
	err := json.Unmarshal(ext.Raw, &res)
	if err != nil {
		return nil, err
	}

	// entry := repo.Entry{
	// 	Name: deriveRepoName(res.Repository),
	// 	URL:  res.Repository,
	// }

	// if r.op != Delete {
	// 	err = r.cli.AddOrUpdateChartRepo(entry)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	timeout := time.Duration(10 * time.Minute)
	if res.WaitTimeout != nil {
		timeout = res.WaitTimeout.Duration
	}

	spec := &helmclient.ChartSpec{
		ReleaseName:     res.Name,
		ChartName:       res.Repository,
		Namespace:       r.ns,
		Version:         res.Version,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		MaxHistory:      ptr.Deref(res.MaxHistory, 10),
		Wait:            ptr.Deref(res.Wait, true),
		ValuesOptions:   r.valuesOptions(id, res.Set),
		Timeout:         timeout,
		Repository:      res.Name,
	}
	if res.InsecureSkipTLSVerify != nil {
		spec.InsecureSkipTLSverify = *res.InsecureSkipTLSVerify
	}
	if res.URL != "" {
		spec.ChartName = res.URL
		spec.ReleaseName = steps.DeriveReleaseName(res.URL)
	}
	if res.ReleaseName != "" {
		spec.ReleaseName = res.ReleaseName
	}

	if res.Credentials != nil {
		secret, err := resolvers.GetSecret(ctx, *r.dyn, res.Credentials.PasswordRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret: %w", err)
		}
		spec.Username = res.Credentials.Username
		spec.Password = secret
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

			r.logr.Debug(fmt.Sprintf(
				"[chart:%s]: set (name: %s, value: %s)",
				id, el.Name, steps.Strval(val)))
		}
	}

	return opts
}
