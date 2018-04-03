package osb

import (
	"fmt"

	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/spf13/viper"
)

type ClientConfiguration struct {
	*osbc.ClientConfiguration
	CreateFunc func(config *osbc.ClientConfiguration) (osbc.Client, error)
}

//TODO  Add TLS config ?
func DefaultConfig() *ClientConfiguration {
	var settings struct {
		User           string
		Password       string
		Host           string
		TimeoutSeconds int
	}

	viperServer := viper.Sub("sm")
	viperServer.SetEnvPrefix("sm")
	viperServer.Unmarshal(&settings)

	clientConfig := osbc.DefaultClientConfiguration()

	clientConfig.Name = "sm"

	if len(settings.Host) != 0 {
		clientConfig.URL = settings.Host + "/osb"
	}

	if len(settings.User) != 0 && len(settings.Password) != 0 {
		clientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: settings.User,
				Password: settings.Password,
			}}
	}
	if settings.TimeoutSeconds != 0 {
		clientConfig.TimeoutSeconds = settings.TimeoutSeconds
	}

	return &ClientConfiguration{
		ClientConfiguration: clientConfig,
		CreateFunc:          osbc.NewClient,
	}
}

func (c *ClientConfiguration) Validate() error {
	if c.CreateFunc == nil {
		return fmt.Errorf("OSB Config error: CreateFunc missing")
	}
	if len(c.URL) == 0 {
		return fmt.Errorf("OSB Config error: URL missing")
	}
	if c.AuthConfig == nil {
		return fmt.Errorf("OSB Config error: AuthConfig missing")
	}
	if c.TimeoutSeconds == 0 {
		return fmt.Errorf("OSB Config error: TimeoutSeconds missing")
	}
	return nil
}
