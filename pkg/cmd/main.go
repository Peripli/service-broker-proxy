package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/cf"
	"github.com/Peripli/service-broker-proxy/pkg/config"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/sirupsen/logrus"
)

//TODO This package should be separate repository (CF Wrapper Module) for the proxy
func main() {
	sbproxyConfig, err := sbproxy.DefaultConfig() //asd
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	osbConfig, err := osb.DefaultConfig()
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	smConfig, err := sm.DefaultConfig()
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	cfConfig, err := cf.DefaultConfig()
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	cfg, err := config.New(sbproxyConfig, osbConfig, smConfig, cfConfig)
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	sbProxy, err := sbproxy.New(cfg)
	if err != nil {
		logrus.Fatal("Error creating SBProxy: ", err)
	}

	sbProxy.Run()
}
