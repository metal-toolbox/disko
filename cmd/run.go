package cmd

import (
	"context"
	"log"
	"strings"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/metal-toolbox/disko/internal/app"
	"github.com/metal-toolbox/disko/internal/metrics"
	"github.com/metal-toolbox/disko/internal/model"
	"github.com/metal-toolbox/disko/internal/store"
	"github.com/metal-toolbox/disko/internal/worker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.hollow.sh/toolbox/events"

	// nolint:gosec // profiling endpoint listens on localhost.
	_ "net/http/pprof"
)

var cmdRun = &cobra.Command{
	Use:   "run",
	Short: "Run disko as a service to listen for events on NATS and mount disk images",
	Run: func(cmd *cobra.Command, args []string) {
		runWorker(cmd.Context())
	},
}

// run worker command
var (
	dryrun       bool
	facilityCode string
	storeKind    string
)

var (
	ErrInventoryStore = errors.New("inventory store error")
)

func runWorker(ctx context.Context) {
	disko, termCh, err := app.New(
		model.AppKindWorker,
		model.StoreKind(storeKind),
		cfgFile,
		logLevel,
		enableProfiling,
	)
	if err != nil {
		log.Fatal(err)
	}

	// serve metrics endpoint
	metrics.ListenAndServe()

	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "disko")
	defer otelShutdown(ctx)

	// Setup cancel context with cancel func.
	ctx, cancelFunc := context.WithCancel(ctx)

	// routine listens for termination signal and cancels the context
	go func() {
		<-termCh
		disko.Logger.Info("got TERM signal, exiting...")
		cancelFunc()
	}()

	stream, err := events.NewStream(*disko.Config.NatsOptions)
	if err != nil {
		disko.Logger.Fatal(err)
	}

	repository, err := initInventory(ctx, disko.Config, disko.Logger)
	if err != nil {
		disko.Logger.Fatal(err)
	}

	w := worker.New(
		facilityCode,
		stream,
		repository,
		disko.Config,
		disko.SyncWg,
		disko.Logger,
	)

	w.Run(ctx)
}

func initInventory(ctx context.Context, config *app.Configuration, logger *logrus.Logger) (store.Repository, error) {
	switch {
	// from CLI flags
	case strings.HasSuffix(storeKind, ".yml"), strings.HasSuffix(storeKind, ".yaml"):
		return store.NewYamlInventory(storeKind)
	case storeKind == string(model.StoreKindServerservice):
		return store.NewServerserviceStore(ctx, config.ServerserviceOptions, logger)
	}

	return nil, errors.Wrap(ErrInventoryStore, "expected a valid inventory store parameter")
}

func init() {
	cmdRun.PersistentFlags().StringVar(&storeKind, "store", "", "Inventory store to lookup devices for update - 'serverservice' or an inventory file with a .yml/.yaml extenstion")
	cmdRun.PersistentFlags().BoolVarP(&dryrun, "dry-run", "", false, "In dryrun mode, the worker actions the task without installing firmware")
	cmdRun.PersistentFlags().StringVar(&facilityCode, "facility-code", "", "The facility code this disko instance is associated with")

	if err := cmdRun.MarkPersistentFlagRequired("store"); err != nil {
		log.Fatal(err)
	}

	rootCmd.AddCommand(cmdRun)
}
