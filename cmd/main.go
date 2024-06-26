package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin/v2"
	"github.com/krateoplatformops/installer/internal/controllers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/krateoplatformops/installer/apis"
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"

	"github.com/stoewer/go-strcase"
)

const (
	providerName = "Installer"
)

func main() {
	envVarPrefix := fmt.Sprintf("%s_PROVIDER", strcase.UpperSnakeCase(providerName))

	var (
		app = kingpin.New(filepath.Base(os.Args[0]), fmt.Sprintf("Krateo %s Provider.", providerName)).
			DefaultEnvars()
		debug = app.Flag("debug", "Run with debug logging.").Short('d').
			Default("true").
			OverrideDefaultFromEnvar(fmt.Sprintf("%s_DEBUG", envVarPrefix)).
			Bool()
		namespace = app.Flag("namespace", "Watch resources only in this namespace.").Short('n').
				OverrideDefaultFromEnvar(fmt.Sprintf("%s_NAMESPACE", envVarPrefix)).
				Default("").String()
		syncPeriod = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').
				Default("1h").
				Duration()
		pollInterval = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").
				Default("5m").
				OverrideDefaultFromEnvar(fmt.Sprintf("%s_POLL_INTERVAL", envVarPrefix)).
				Duration()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").
					Default("3").
					OverrideDefaultFromEnvar(fmt.Sprintf("%s_MAX_RECONCILE_RATE", envVarPrefix)).
					Int()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").
				Short('l').
				Default("false").
				OverrideDefaultFromEnvar(fmt.Sprintf("%s_LEADER_ELECTION", envVarPrefix)).
				Bool()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	log.Default().SetOutput(io.Discard)
	ctrl.SetLogger(zap.New(zap.WriteTo(io.Discard)))

	zl := zap.New(zap.UseDevMode(*debug))
	logr := logging.NewLogrLogger(zl.WithName(fmt.Sprintf("%s-provider", strcase.KebabCase(providerName))))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	logr.Debug("Starting", "sync-period", syncPeriod.String(), "poll-interval", pollInterval.String())

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	co := cache.Options{
		SyncPeriod: syncPeriod,
	}
	if len(*namespace) > 0 {
		co.DefaultNamespaces = map[string]cache.Config{
			*namespace: {},
		}
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: fmt.Sprintf("leader-election-%s-provider", strcase.KebabCase(providerName)),
		Cache:            co,
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	o := controller.Options{
		Logger:                  logr,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
	}

	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add APIs to scheme")
	kingpin.FatalIfError(controllers.Setup(mgr, o), "Cannot setup controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
