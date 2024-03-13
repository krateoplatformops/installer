package release

import (
	"context"
	"fmt"
	"time"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/helmclient"
	"github.com/krateoplatformops/installer/internal/helmclient/values"
	"github.com/krateoplatformops/installer/internal/kubernetes/secrets"
	"github.com/krateoplatformops/installer/internal/kubernetes/services"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

const (
	defaultServiceType = "NodePort"
)

func newInstaller(rc *rest.Config, cr *v1alpha1.KrateoPlatformOps, log logging.Logger) (*installer, error) {
	helmClient, err := newHelmClient(helmClientOptions{
		namespace:  cr.GetNamespace(),
		restConfig: rc,
		log:        log,
		verbose:    meta.IsVerbose(cr),
	})
	if err != nil {
		return nil, err
	}

	secretsClient, err := secrets.NewClient(rc)
	if err != nil {
		return nil, err
	}

	servicesClient, err := services.NewClient(rc)
	if err != nil {
		return nil, err
	}

	return &installer{
		helmClient:     helmClient,
		servicesClient: servicesClient,
		secretsClient:  secretsClient,
		cr:             cr.DeepCopy(),
	}, nil
}

type installer struct {
	helmClient     helmclient.Client
	servicesClient *services.Client
	secretsClient  *secrets.Client
	cr             *v1alpha1.KrateoPlatformOps
}

func (i *installer) install(ctx context.Context) error {

	for _, x := range i.cr.Spec.Releases {
		entry := repo.Entry{
			Name: deriveRepoName(x.Repository),
			URL:  x.Repository,
		}
		err := i.helmClient.AddOrUpdateChartRepo(entry)
		if err != nil {
			return err
		}

		if isKrateoGateway(entry.Name) {
			err = i.handleKrateoGateway(ctx, entry, x)
			if err != nil {
				return err
			}
			continue
		}

		err = i.handleGenericChart(ctx, entry, x)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *installer) handleKrateoGateway(ctx context.Context, entry repo.Entry, params *v1alpha1.ReleaseParameters) error {

	return nil
}

func (i *installer) handleGenericChart(ctx context.Context, entry repo.Entry, params *v1alpha1.ReleaseParameters) error {
	serviceType := ptr.Deref(i.cr.Spec.ServiceType, defaultServiceType)

	vals := []string{fmt.Sprintf("service.type=%s", serviceType)}
	for _, y := range params.Set {
		vals = append(vals, fmt.Sprintf("%s=%s", y.Name, y.Value))
	}

	chartSpec := helmclient.ChartSpec{
		ReleaseName:     params.Name,
		ChartName:       fmt.Sprintf("%s/%s", entry.Name, params.Name),
		Namespace:       i.cr.GetNamespace(),
		Version:         params.Version,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		Wait:            true,
		ValuesOptions:   values.Options{Values: vals},
		Timeout:         time.Duration(10 * time.Minute),
	}

	_, err := i.helmClient.InstallOrUpgradeChart(ctx, &chartSpec, nil)

	return err
}
