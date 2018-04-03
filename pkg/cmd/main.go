package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/cf"
	"github.com/Peripli/service-broker-proxy/pkg/config"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/sirupsen/logrus"
)

//TODO This package should be seperate repository (CF Wrapper Module) for the proxy
func main() {
	cfg, err := config.New(
		sbproxy.DefaultConfig(),
		osb.DefaultConfig(),
		sm.DefaultConfig(),
		cf.DefaultConfig(),
	)
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	sbProxy, err := sbproxy.New(cfg)
	if err != nil {
		logrus.Fatal("Error creating SBProxy: ", err)
	}

	sbProxy.Run()
}
