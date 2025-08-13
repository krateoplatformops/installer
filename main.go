package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-logr/logr"

	"github.com/krateoplatformops/installer/internal/controllers"
	"github.com/krateoplatformops/plumbing/env"
	prettylog "github.com/krateoplatformops/plumbing/slogs/pretty"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"

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

	debug := flag.Bool("debug", env.Bool(fmt.Sprintf("%s_DEBUG", envVarPrefix), false), "Run with debug logging.")
	namespace := flag.String("namespace", env.String(fmt.Sprintf("%s_NAMESPACE", envVarPrefix), ""), "Watch resources only in this namespace.")
	syncPeriod := flag.Duration("sync", env.Duration(fmt.Sprintf("%s_SYNC", envVarPrefix), time.Hour*1), "Controller manager sync period such as 300ms, 1.5h, or 2h45m")
	pollInterval := flag.Duration("poll", env.Duration(fmt.Sprintf("%s_POLL_INTERVAL", envVarPrefix), time.Minute*5), "Poll interval controls how often an individual resource should be checked for drift.")
	maxReconcileRate := flag.Int("max-reconcile-rate", env.Int(fmt.Sprintf("%s_MAX_RECONCILE_RATE", envVarPrefix), 3), "The global maximum rate per second at which resources may checked for drift from the desired state.")
	leaderElection := flag.Bool("leader-election", env.Bool(fmt.Sprintf("%s_LEADER_ELECTION", envVarPrefix), false), "Use leader election for the controller manager.")
	maxErrorRetryInterval := flag.Duration("max-error-retry-interval", env.Duration(fmt.Sprintf("%s_MAX_ERROR_RETRY_INTERVAL", envVarPrefix), 0*time.Minute), "The maximum interval between retries when an error occurs. This should be less than the half of the poll interval.")
	minErrorRetryInterval := flag.Duration("min-error-retry-interval", env.Duration(fmt.Sprintf("%s_MIN_ERROR_RETRY_INTERVAL", envVarPrefix), 1*time.Second), "The minimum interval between retries when an error occurs. This should be less than max-error-retry-interval.")

	flag.Parse()

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}

	lh := prettylog.New(&slog.HandlerOptions{
		Level:     logLevel,
		AddSource: false,
	},
		prettylog.WithDestinationWriter(os.Stderr),
		prettylog.WithColor(),
		prettylog.WithOutputEmptyAttrs(),
	)

	logrlog := logr.FromSlogHandler(slog.New(lh).Handler())
	log := logging.NewLogrLogger(logrlog)

	log.Info("Starting", "sync-period", syncPeriod.String())

	if maxErrorRetryInterval.Seconds() == 0 {
		retryInterval := (*pollInterval / 2)
		maxErrorRetryInterval = &retryInterval
	} else if maxErrorRetryInterval.Seconds() >= pollInterval.Seconds() {
		retryInterval := (*pollInterval / 2)
		maxErrorRetryInterval = &retryInterval

		log.Info("[WARNING] max-error-retry-interval is greater than or equal to poll interval, setting to half of poll interval", "max-error-retry-interval", maxErrorRetryInterval.String())
	}

	if minErrorRetryInterval.Seconds() >= maxErrorRetryInterval.Seconds() {
		retryInterval := 1 * time.Second
		minErrorRetryInterval = &retryInterval

		log.Info("[WARNING] min-error-retry-interval is greater than or equal to max-error-retry-interval, setting to 1 second", "min-error-retry-interval", minErrorRetryInterval.String())
	}

	log.Debug("Starting", "sync-period", syncPeriod.String(), "poll-interval", pollInterval.String(), "max-error-retry-interval", maxErrorRetryInterval.String())

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Info("Cannot get API server rest config", "error", err.Error())
		os.Exit(1)
	}

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
	if err != nil {
		log.Info("Cannot create controller manager, continuing", "error", err.Error())
		os.Exit(1)
	}

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobalExponential(*minErrorRetryInterval, *maxErrorRetryInterval),
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Info("Cannot add APIs to scheme, continuing", "error", err.Error())
		os.Exit(1)
	}
	if err := controllers.Setup(mgr, o); err != nil {
		log.Info("Cannot setup controllers, continuing", "error", err.Error())
		os.Exit(1)
	}
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Info("Cannot start controller manager, continuing", "error", err.Error())
		os.Exit(1)
	}
}
