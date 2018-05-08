package sbproxy

import (
	"context"
	"strconv"
	"sync"

	"os"
	"os/signal"

	"time"

	"net/http"

	"github.com/Peripli/service-broker-proxy/pkg/logger"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/middleware"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/gorilla/mux"
	"github.com/onrik/logrus/filename"
	"github.com/onrik/logrus/formatter"
	"github.com/pkg/errors"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	"github.com/pmorie/osb-broker-lib/pkg/rest"
	osbserver "github.com/pmorie/osb-broker-lib/pkg/server"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

const (
	// BrokerPathParam
	BrokerPathParam = "brokerID"

	// APIPrefix
	APIPrefix = "/v1/osb"

	// Path
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

var (
	group sync.WaitGroup
)

// SBProxy type is the starting point of the proxy application. It glues the proxy REST API and the timed
// jobs for broker registrations
type SBProxy struct {
	CronScheduler *cron.Cron
	Server        *osbserver.Server
	AppConfig     *server.AppConfiguration
}

// New builds a new SBProxy from the provided configuration using the provided platform client. The
// platform client is used by the SBProxy to call to the platform during broker creation and deletion.
func New(config *Configuration, client platform.Client) (*SBProxy, error) {

	if err := config.Validate(); err != nil {
		return nil, err
	}

	appConfig := config.App
	setUpLogging(appConfig.LogLevel, appConfig.LogFormat)

	osbServer, err := defaultOSBServer(config.Osb)
	if err != nil {
		return nil, err
	}

	osbServer.Router.Use(middleware.LogRequest())

	cronScheduler := cron.New()

	regJob, err := defaultRegJob(&group, client, config.Sm, config.App.Host+APIPrefix)
	if err != nil {
		return nil, err
	}

	if err := cronScheduler.AddJob("@every 1m", regJob); err != nil {
		return nil, errors.Wrap(err, "error adding registration job")
	}

	return &SBProxy{
		Server:        osbServer,
		CronScheduler: cronScheduler,
		AppConfig:     appConfig,
	}, nil
}

// Use provides a way to plugin middleware in the SBProxy
func (s SBProxy) Use(middleware func(handler http.Handler) http.Handler) {
	s.Server.Router.Use(middleware)
}

// Run is the entrypoint of the SBProxy. Run boots the application.
func (s SBProxy) Run() {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer waitWithTimeout(&group, time.Duration(s.AppConfig.TimeoutSec)*time.Second)
	defer cancel()

	handleInterrupts(ctx, cancel)

	s.CronScheduler.Start()
	defer s.CronScheduler.Stop()

	addr := ":" + strconv.Itoa(s.AppConfig.Port)

	logrus.Info("Running SBProxy server...")
	if s.AppConfig.TLSKey != "" && s.AppConfig.TLSCert != "" {
		err = s.Server.RunTLS(ctx, addr, s.AppConfig.TLSCert, s.AppConfig.TLSKey)
	} else {
		err = s.Server.Run(ctx, addr)
	}
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		logrus.WithError(errors.WithStack(err)).Errorln("Error occurred while sbproxy was running")
	}
}

// AddJob provides a way to add additional cron jobs to run alongside the SBProxy application.
func (s *SBProxy) AddJob(schedule string, job cron.Job) {
	s.CronScheduler.AddJob(schedule, job)
}

func defaultOSBServer(config *osb.ClientConfiguration) (*osbserver.Server, error) {
	businessLogic, err := osb.NewBusinessLogic(config)
	if err != nil {
		return nil, err
	}

	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := rest.NewAPISurface(businessLogic, osbMetrics)
	if err != nil {
		return nil, errors.Wrap(err, "error creating OSB API surface")
	}

	osbServer := osbserver.New(api, reg)
	router := mux.NewRouter()

	err = registerRoutes(Path, osbServer.Router, router)
	if err != nil {
		return nil, err
	}

	osbServer.Router = router
	return osbServer, nil
}

func registerRoutes(prefix string, fromRouter *mux.Router, toRouter *mux.Router) error {
	subRouter := toRouter.PathPrefix(prefix).Subrouter()
	return fromRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {

		path, err := route.GetPathTemplate()
		if err != nil {
			return errors.Wrap(err, "error getting path template")
		}

		methods, err := route.GetMethods()
		if err != nil {
			return errors.Wrap(err, "error getting route methods")
		}
		logrus.Info("Registering route with methods: ", methods, " and path: ", path)
		subRouter.Handle(path, route.GetHandler()).Methods(methods...)
		return nil
	})
}

func defaultRegJob(group *sync.WaitGroup, platformClient platform.Client, smConfig *sm.ClientConfiguration, proxyHost string) (cron.Job, error) {
	smClient, err := smConfig.CreateFunc(smConfig)
	if err != nil {
		return nil, err
	}
	regTask := NewTask(group, platformClient, smClient, proxyHost)

	return regTask, nil
}

//TODO: should happen earlier (ideally in sbproxy init(), logger.DefaultConfig()?)
func setUpLogging(logLevel string, logFormat string) {
	logrus.AddHook(&logger.ErrorLocationHook{})
	hook := filename.NewHook()
	hook.Field = "logsource"
	logrus.AddHook(hook)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.WithError(err).Debug("Could not parse log level configuration")
	} else {
		logrus.SetLevel(level)
	}
	if logFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		textFormatter := formatter.New()
		logrus.SetFormatter(textFormatter)
	}
}

// handleInterrupts hannles OS interrupt signals by canceling the context
func handleInterrupts(ctx context.Context, cancel context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancel()
		case <-ctx.Done():
			return
		}
	}()
}

// waitWithTimeout waits for a WaitGroup to finish for a certain duration and times out afterwards
// WaitGroup parameter should be pointer or else the copy won't get notified about .Done() calls
func waitWithTimeout(group *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		group.Wait()
	}()
	select {
	case <-c:
		logrus.Debug("Timeout WaitGroup ", group, " finished successfully")
	case <-time.After(timeout):
		logrus.Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
