package app

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/metal-toolbox/disko/internal/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	ErrAppInit = errors.New("error initializing app")
)

const (
	ProfilingEndpoint = "localhost:9091"
)

// App holds attributes for running disko.
type App struct {
	// Viper loads configuration parameters.
	v *viper.Viper

	// App configuration.
	Config *Configuration

	// TermCh is the channel to terminate the app based on a signal
	TermCh chan os.Signal

	// Sync waitgroup to wait for running go routines on termination.
	SyncWg *sync.WaitGroup

	// Logger is the app logger
	Logger *logrus.Logger

	// Kind is the type of application - inband/outofband
	Kind model.AppKind
}

// New returns a new disko application object with the configuration loaded
func New(appKind model.AppKind, storeKind model.StoreKind, cfgFile, loglevel string, profiling bool) (*App, <-chan os.Signal, error) {
	switch appKind {
	case model.AppKindClient, model.AppKindWorker:
	default:
		return nil, nil, errors.Wrap(ErrAppInit, "invalid app kind: "+string(appKind))
	}

	app := &App{
		v:      viper.New(),
		Kind:   appKind,
		Config: &Configuration{},
		TermCh: make(chan os.Signal),
		SyncWg: &sync.WaitGroup{},
		Logger: logrus.New(),
	}

	if err := app.LoadConfiguration(cfgFile, storeKind); err != nil {
		return nil, nil, err
	}

	switch model.LogLevel(loglevel) {
	case model.LogLevelDebug:
		app.Logger.Level = logrus.DebugLevel
	case model.LogLevelTrace:
		app.Logger.Level = logrus.TraceLevel
	default:
		app.Logger.Level = logrus.InfoLevel
	}

	app.Logger.SetFormatter(&logrus.JSONFormatter{})

	// register for SIGINT, SIGTERM
	signal.Notify(app.TermCh, syscall.SIGINT, syscall.SIGTERM)

	if profiling {
		enableProfilingEndpoint()
	}

	termCh := make(chan os.Signal, 1)

	return app, termCh, nil
}

// enableProfilingEndpoint enables the profiling endpoint
func enableProfilingEndpoint() {
	go func() {
		server := &http.Server{
			Addr:              ProfilingEndpoint,
			ReadHeaderTimeout: 2 * time.Second, // nolint:gomnd // time duration value is clear as is.
		}

		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	log.Println("profiling enabled: " + ProfilingEndpoint + "/debug/pprof")
}

// NewLogrusEntryFromLogger returns a logger contextualized with the given logrus fields.
func NewLogrusEntryFromLogger(fields logrus.Fields, logger *logrus.Logger) *logrus.Entry {
	l := logrus.New()
	l.Formatter = logger.Formatter
	loggerEntry := logger.WithFields(fields)
	loggerEntry.Level = logger.Level

	return loggerEntry
}
