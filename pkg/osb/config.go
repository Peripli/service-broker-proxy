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

//TODO combine these settings with sm config somehow?
type settings struct {
	User           string
	Password       string
	Host           string
	TimeoutSeconds int
}

func DefaultConfig() (*ClientConfiguration, error) {
	osbConfig := &struct {
		Sm *settings
	}{
		Sm: &settings{},
	}
	//TODO BindEnv
	if err := viper.Unmarshal(osbConfig); err != nil {
		return nil, err
	}

	clientConfig := osbc.DefaultClientConfiguration()
	clientConfig.Name = "sm"

	if len(osbConfig.Sm.Host) != 0 {
		clientConfig.URL = osbConfig.Sm.Host + "/osb"
	}

	if len(osbConfig.Sm.User) != 0 && len(osbConfig.Sm.Password) != 0 {
		clientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: osbConfig.Sm.User,
				Password: osbConfig.Sm.Password,
			}}
	}
	if osbConfig.Sm.TimeoutSeconds != 0 {
		clientConfig.TimeoutSeconds = osbConfig.Sm.TimeoutSeconds
	}

	return &ClientConfiguration{
		ClientConfiguration: clientConfig,
		CreateFunc:          osbc.NewClient,
	}, nil
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
