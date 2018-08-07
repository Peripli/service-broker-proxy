package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-broker-proxy/pkg/middleware"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"

	// Path for the Proxy OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

// Server type is the starting point of the proxy application. It glues the proxy REST API and the timed
// jobs for broker registrations
type Server struct {
	router *mux.Router

	Config *Config
}

// New builds a new Server from the provided configuration using the provided platform client. The
// platform client is used by the Server to call to the platform during broker creation and deletion.
func New(config *Config, osbConfig *osb.ClientConfig) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if err := osbConfig.Validate(); err != nil {
		return nil, err
	}

	logging.Setup(config.LogLevel, config.LogFormat)

	router, err := osbRouter(osbConfig)
	if err != nil {
		return nil, err
	}
	router.Use(middleware.LogRequest())

	return &Server{
		router: router,
		Config: config,
	}, nil
}

func (s *Server) Use(middleware func(handler http.Handler) http.Handler) {
	s.router.Use(middleware)
}

func (s *Server) run(ctx context.Context, addr string, listenAndServe func(srv *http.Server) error) error {
	glog.Infof("Starting server on %s\n", addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if srv.Shutdown(c) != nil {
			srv.Close()
		}
	}()
	return listenAndServe(srv)
}

// Run is the entrypoint of the Server. Run boots the application.
func (s Server) Run(group *sync.WaitGroup) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer waitWithTimeout(group, s.Config.Timeout)
	defer cancel()

	handleInterrupts(ctx, cancel)

	addr := ":" + strconv.Itoa(s.Config.Port)

	logrus.Info("Running Server...")
	// if s.Config.TLSKey != "" && s.Config.TLSCert != "" {
	// 	err = s.Server.RunTLS(ctx, addr, s.Config.TLSCert, s.Config.TLSKey)
	// } else {
	// 	err = s.Server.Run(ctx, addr)
	// }
	listenAndServe := func(srv *http.Server) error {
		return srv.ListenAndServe()
	}
	err = s.run(ctx, addr, listenAndServe)

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		logrus.WithError(errors.WithStack(err)).Errorln("Error occurred while sbproxy was running")
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

func osbRouter(config *osb.ClientConfig) (*mux.Router, error) {
	businessLogic, err := osb.NewBusinessLogic(config)
	if err != nil {
		return nil, err
	}

	// reg := prom.NewRegistry()
	// osbMetrics := metrics.New()
	// reg.MustRegister(osbMetrics)

	// api, err := rest.NewAPISurface(businessLogic, osbMetrics)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "error creating OSB API surface")
	// }

	// osbServer := osbserver.New(api, reg)

	router := mux.NewRouter()
	router.HandleFunc("/v2/catalog", businessLogic.HandleRequest).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", businessLogic.HandleRequest).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", businessLogic.HandleRequest).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}", businessLogic.HandleRequest).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}", businessLogic.HandleRequest).Methods("PATCH")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", businessLogic.HandleRequest).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", businessLogic.HandleRequest).Methods("DELETE")

	return router, nil

	// err = registerRoutes(Path, osbServer.Router, router)
	// if err != nil {
	// 	return nil, err
	// }

	// osbServer.Router = router
	// return osbServer, nil
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
		logrus.Info("Registering route with methods: ", methods, " and path: ", prefix+path)
		subRouter.Handle(path, route.GetHandler()).Methods(methods...)
		return nil
	})
}
