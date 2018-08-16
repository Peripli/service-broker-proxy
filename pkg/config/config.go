package config

import (
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/spf13/pflag"
)

// Settings type holds all config properties for the sbproxy
type Settings struct {
	Server   *server.Settings `mapstructure:"server"`
	Log      *log.Settings    `mapstructure:"log"`
	Sm       *sm.Settings     `mapstructure:"sm"`
	SelfHost string           `mapstructure:"self_host"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Server:   server.DefaultSettings(),
		Log:      log.DefaultSettings(),
		Sm:       sm.DefaultSettings(),
		SelfHost: "",
	}
}

func NewSettings(env env.Environment) (*Settings, error) {
	config := DefaultSettings()
	if err := env.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

// AddPFlags adds the SM config flags to the provided flag set
func AddPFlags(set *pflag.FlagSet) {
	env.CreatePFlags(set, DefaultSettings())

	env.CreatePFlagsForConfigFile(set)
}

// New builds an config.Settings from the specified Environment
func New(env env.Environment) (*Settings, error) {
	// config, err :=
	return NewSettings(env)

	// return &Settings{
	// 	Server: serverConfig,
	// 	Log:    logConfig,
	// 	Sm:     smConfig,
	// }, nil
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Settings) Validate() error {
	validatable := []interface{ Validate() error }{c.Server, c.Sm, c.Log}

	for _, item := range validatable {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
