package controllers

import (
	"log"
	"os"

	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/krateoplatformops/installer/internal/controllers/workflows"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	log.SetOutput(os.Stderr)

	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		workflows.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
