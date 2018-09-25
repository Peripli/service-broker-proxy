package sbproxy

import (
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/spf13/pflag"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
)

type validatable interface {
	Validate() error
}

// Settings type holds all config properties for the sbproxy
type Settings struct {
	Server    *server.Settings    `mapstructure:"server"`
	Log       *log.Settings       `mapstructure:"log"`
	Sm        *sm.Settings        `mapstructure:"sm"`
	Reconcile *reconcile.Settings `mapstructure:"self"`
}

// DefaultSettings returns default value for the proxy settings
func DefaultSettings() *Settings {
	return &Settings{
		Server:    server.DefaultSettings(),
		Log:       log.DefaultSettings(),
		Sm:        sm.DefaultSettings(),
		Reconcile: reconcile.DefaultSettings(),
	}
}

// NewSettings creates new proxy settings from the specified environment
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

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	validatable := []validatable {c.Server, c.Log, c.Sm, c.Reconcile}

	for _, item := range validatable {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
