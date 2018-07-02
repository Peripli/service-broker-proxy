package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/golang/glog"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	"github.com/pmorie/osb-broker-lib/pkg/rest"
	"github.com/pmorie/osb-broker-lib/pkg/server"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func main12456() {
	//make platform client
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		glog.Fatalln(err)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	// if listen and serve crashes this guy will let everybody who is waiting for signals to know to stop waiting
	defer cancel()

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Info("Received OS interrupt, exiting gracefully...")
			cancel()
		case <-ctx.Done():
		}
	}()

	return runServer(ctx)
}

func runServer(ctx context.Context) error {
	config := parseConfiguration()

	addr := ":" + strconv.Itoa(config.ServerPort)

	businessLogic, err := osb.NewBusinessLogic(nil)
	if err != nil {
		return err
	}

	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := rest.NewAPISurface(businessLogic, osbMetrics)
	if err != nil {
		return err
	}

	s := server.New(api, reg)

	logrus.Info("Starting osb!")

	return s.Run(ctx, addr)
}
