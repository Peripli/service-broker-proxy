package config

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type configuration struct {
	Sbproxy  *sbproxy.ServerConfiguration
	Osb      *osb.ClientConfiguration
	Sm       *sm.ClientConfiguration
	Platform platform.ClientConfiguration
}

var _ sbproxy.Configuration = &configuration{}

func (c *configuration) SbproxyConfig() *sbproxy.ServerConfiguration {
	return c.Sbproxy
}

func (c *configuration) OsbConfig() *osb.ClientConfiguration {
	return c.Osb
}

func (c *configuration) SmConfig() *sm.ClientConfiguration {
	return c.Sm
}

func (c *configuration) PlatformConfig() platform.ClientConfiguration {
	return c.Platform
}

func (c *configuration) Validate() error {

	if err := c.Sbproxy.Validate(); err != nil {
		return err
	}
	if err := c.Osb.Validate(); err != nil {
		return err
	}
	if err := c.Sm.Validate(); err != nil {
		return err
	}
	return c.Platform.Validate()
}

func init() {
	viper.SetConfigName("application")
	viper.SetConfigType("yml")
	viper.AddConfigPath("../../../github.com/Peripli/service-broker-proxy")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal("Failed to read the configuration file: ", err)
	}
}

func New(
	SbproxyConfig *sbproxy.ServerConfiguration,
	OsbConfig *osb.ClientConfiguration,
	SmConfig *sm.ClientConfiguration,
	PlatformConfig platform.ClientConfiguration,
) (*configuration, error) {
	config := &configuration{
		Sbproxy:  SbproxyConfig,
		Osb:      OsbConfig,
		Sm:       SmConfig,
		Platform: PlatformConfig,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}
