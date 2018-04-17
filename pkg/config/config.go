package config

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//TODO: abstract logic so that different file names and locations can be used
func init() {
	viper.SetConfigName("application")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
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

	return config, nil
}

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
		return fmt.Errorf("config validation error: %s", err)
	}
	if err := c.Osb.Validate(); err != nil {
		return fmt.Errorf("config validation error: %s", err)
	}
	if err := c.Sm.Validate(); err != nil {
		return fmt.Errorf("config validation error: %s", err)
	}
	if err := c.Platform.Validate(); err != nil {
		return fmt.Errorf("config validation error: %s", err)
	}
	return nil
}
