package sbproxy

import (
	"context"
	"strconv"
	"sync"

	"os"
	"os/signal"

	"time"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/middleware"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/task"
	"github.com/gorilla/mux"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	"github.com/pmorie/osb-broker-lib/pkg/rest"
	"github.com/pmorie/osb-broker-lib/pkg/server"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"fmt"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
	group  sync.WaitGroup
)

type Configuration interface {

	Validate() error
	SbproxyConfig() *ServerConfiguration
	OsbConfig() *osb.ClientConfiguration
	SmConfig() *sm.ClientConfiguration
	PlatformConfig() platform.ClientConfiguration
}

type SBProxy struct {
	CronScheduler *cron.Cron
	Server        *server.Server
	ServerConfig  *ServerConfiguration
}

func New(config Configuration) (*SBProxy, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if  err := config.Validate(); err != nil {
		return nil, err
	}

	sbproxyConfig := config.SbproxyConfig()
	setUpLogging(sbproxyConfig.LogLevel, sbproxyConfig.LogFormat)

	osbServer, err := defaultOSBServer(config.OsbConfig())
	if err != nil {
		return nil, err
	}

	details := sbproxyConfig.Reg
	osbServer.Router.Use(middleware.LogRequest())
	osbServer.Router.Use(middleware.BasicAuth(details.User, details.Password))

	cronScheduler := cron.New()
	regJob, err := defaultRegJob(config.PlatformConfig(), config.SmConfig(), details)
	if err != nil {
		return nil, err
	}
	cronScheduler.AddJob("@every 1m", regJob)

	if err != nil {
		return nil, err
	}

	return &SBProxy{
		Server:        osbServer,
		CronScheduler: cronScheduler,
	}, nil
}

func (s SBProxy) Run() {
	var err error
	ctx, cancel = context.WithCancel(context.Background())
	defer waitWithTimeout(&group, time.Duration(s.ServerConfig.TimeoutSec)*time.Second)
	defer cancel()

	handleInterrupts()

	s.CronScheduler.Start()
	defer s.CronScheduler.Stop()

	addr := ":" + strconv.Itoa(s.ServerConfig.Port)

	if s.ServerConfig.TLSKey != "" && s.ServerConfig.TLSCert != "" {
		err = s.Server.RunTLS(ctx, addr, s.ServerConfig.TLSCert, s.ServerConfig.TLSKey)
	} else {
		err = s.Server.Run(ctx, addr)
	}
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		logrus.Errorln("Error occurred while sbproxy was running: ", err)
	}
}

func (s *SBProxy) AddJob(schedule string, job cron.Job) {
	s.CronScheduler.AddJob(schedule, job)
}

func defaultOSBServer(config *osb.ClientConfiguration) (*server.Server, error) {
	businessLogic, err := osb.NewBusinessLogic(config)
	if err != nil {
		return nil, err
	}

	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := rest.NewAPISurface(businessLogic, osbMetrics)
	if err != nil {
		return nil, err
	}

	osbServer := server.New(api, reg)
	router := mux.NewRouter()

	err = moveRoutes("/{brokerID}", osbServer.Router, router)
	if err != nil {
		return nil, err
	}

	osbServer.Router = router
	return osbServer, nil
}

func moveRoutes(prefix string, fromRouter *mux.Router, toRouter *mux.Router) error {
	subRouter := toRouter.PathPrefix(prefix).Subrouter()
	return fromRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {

		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}

		methods, err := route.GetMethods()
		if err != nil {
			return err
		}
		logrus.Info("Adding route with methods: ", methods, " and path: ", path)
		subRouter.Handle(path, route.GetHandler()).Methods(methods...)
		return nil
	})
}

func defaultRegJob(platformConfig platform.ClientConfiguration, smConfig *sm.ClientConfiguration, details *task.RegistrationDetails) (cron.Job, error) {
	platformClient, err := platformConfig.CreateFunc()
	if err != nil {
		return nil, err
	}
	smClient, err := smConfig.CreateFunc(smConfig)
	if err != nil {
		return nil, err
	}
	regTask := task.New(&group, platformClient, smClient, details)

	return regTask, nil
}

//TODO: Should be more generic, log configuration should accept additional params to allow Kibana-styled logs
func setUpLogging(logLevel string, logFormat string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal("Could not parse log level configuration")
	}
	logrus.SetLevel(level)
	if logFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
}

// handleInterrupts hannles OS interrupt signals by canceling the context
func handleInterrupts() {
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
		logrus.Fatal("Shutdown took more than ", timeout)
	case <-time.After(timeout):
		logrus.Debug("Timeout WaitGroup ", group, " finished successfully")
		close(c)
	}
	}
