// Package osb contains logic for building the Service Broker Proxy OSB API
package osb

import (
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

// ClientConfiguration type holds config info for building an OSB client
type ClientConfiguration struct {
	*osbc.ClientConfiguration
	CreateFunc func(config *osbc.ClientConfiguration) (osbc.Client, error)
}

// NewConfig creates ClientConfiguration from the provided settings
func NewConfig(settings *sm.ClientConfiguration) (*ClientConfiguration, error) {

	clientConfig := osbc.DefaultClientConfiguration()
	clientConfig.Name = "sbproxy"

	if len(settings.Host) != 0 {
		clientConfig.URL = settings.Host + settings.OsbAPI
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
	}, nil
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfiguration) Validate() error {
	if c.CreateFunc == nil {
		return errors.New("OSB client configuration CreateFunc missing")
	}
	if c.ClientConfiguration == nil {
		return errors.New("OSB client configuration missing")
	}
	if len(c.URL) == 0 {
		return errors.New("OSB client configuration URL missing")
	}
	if c.AuthConfig == nil {
		return errors.New("OSB client configuration AuthConfig missing")
	}
	if c.TimeoutSeconds == 0 {
		return errors.New("OSB client configuration TimeoutSeconds missing")
	}
	return nil
}
