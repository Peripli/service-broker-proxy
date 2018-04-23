package sbproxy

import (
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
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
		logrus.WithError(errors.WithStack(err)).Fatal("Failed to read the Configuration file")
	}
}

func NewConfig(
	AppConfig *server.AppConfiguration,
	OsbConfig *osb.ClientConfiguration,
	SmConfig *sm.ClientConfiguration,
	PlatformConfig platform.ClientConfiguration,
) (*Configuration, error) {
	config := &Configuration{
		App:      AppConfig,
		Osb:      OsbConfig,
		Sm:       SmConfig,
		Platform: PlatformConfig,
	}

	return config, nil
}

type Configuration struct {
	App      *server.AppConfiguration
	Osb      *osb.ClientConfiguration
	Sm       *sm.ClientConfiguration
	Platform platform.ClientConfiguration
}

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
	if err := c.Platform.Validate(); err != nil {
		return err
	}
	return nil
}
