package sbproxy

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
)

// NewConfigFromEnv builds an sbproxy.Configuration from the specified Environment
func NewConfigFromEnv(env env.Environment) (*Configuration, error) {
	appConfig, err := server.NewConfig(env)
	if err != nil {
		return nil, err
	}
	smConfig, err := sm.NewConfig(env)
	if err != nil {
		return nil, err
	}
	osbConfig, err := osb.NewConfig(smConfig)
	if err != nil {
		return nil, err
	}

	config := &Configuration{
		App: appConfig,
		Osb: osbConfig,
		Sm:  smConfig,
	}

	return config, nil
}

// Configuration type holds all config properties for the sbproxy
type Configuration struct {
	App *server.AppConfiguration
	Osb *osb.ClientConfiguration
	Sm  *sm.ClientConfiguration
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Configuration) Validate() error {

	if err := c.App.Validate(); err != nil {
		return err
	}
	if err := c.Osb.Validate(); err != nil {
		return err
	}
	if err := c.Sm.Validate(); err != nil {
		return err
	}
	return nil
}
