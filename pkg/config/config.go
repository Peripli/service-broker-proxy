package config

import (
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/spf13/pflag"
	"github.com/Peripli/service-manager/pkg/log"
)

// Settings type holds all config properties for the sbproxy
type Settings struct {
	Server  *server.Settings
	Log     *log.Settings
	Sm     *sm.Settings
}

// AddPFlags adds the SM config flags to the provided flag set
func AddPFlags(set *pflag.FlagSet) {
	env.CreatePFlags(set, server.DefaultSettings())
	env.CreatePFlags(set, log.DefaultSettings())
	env.CreatePFlags(set, sm.DefaultSettings())

	env.CreatePFlagsForConfigFile(set)
}


// New builds an config.Settings from the specified Environment
func New(env env.Environment) (*Settings, error) {
	serverConfig, err := server.NewSettings(env)
	if err != nil {
		return nil, err
	}

	logConfig, err := log.NewSettings(env)
	if err != nil {
		return nil, err
	}

	smConfig, err := sm.NewSettings(env)
	if err != nil {
		return nil, err
	}

	return &Settings{
		Server: serverConfig,
		Log:    logConfig,
		Sm:     smConfig,
	}, nil
}


// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Settings) Validate() error {
	validatable := []interface{ Validate() error }{c.Server, c.Sm}

	for _, item := range validatable {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
