package cf

import (
	"fmt"
	"net/http"

	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/spf13/viper"
)

//TODO This package should be in seperate repository (CF Wrapper Module) for the proxy
type PlatformClientConfiguration struct {
	*cfclient.Config
	createFunc func(*cfclient.Config) (*cfclient.Client, error)
}

var _ platform.ClientConfiguration = &PlatformClientConfiguration{}

func (c *PlatformClientConfiguration) CreateFunc() (platform.Client, error) {
	return NewClient(c)
}

func (c *PlatformClientConfiguration) Validate() error {
	if c.createFunc == nil {
		return fmt.Errorf("Platform config error: createFunc missing")
	}
	if c.Config == nil {
		return fmt.Errorf("Platform config error: CF config missing")
	}
	if len(c.Config.ApiAddress) == 0 {
		return fmt.Errorf("Platform config error: CF ApiAddress missing")
	}
	if len(c.Config.ClientID) == 0 {
		return fmt.Errorf("Platform config error: CF ClientID missing")
	}
	if len(c.Config.ClientSecret) == 0 {
		return fmt.Errorf("Platform config error: CF ClientSecret missing")
	}
	return nil
}

func DefaultConfig() *PlatformClientConfiguration {
	var settings struct {
		Api            string
		ClientID       string
		ClientSecret   string
		SkipSSLVerify  bool
		TimeoutSeconds int
	}
	viperServer := viper.Sub("cf")
	viperServer.SetEnvPrefix("cf")
	viperServer.Unmarshal(&settings)

	clientConfig := cfclient.DefaultConfig()

	if len(settings.Api) != 0 {
		clientConfig.ApiAddress = settings.Api
	}
	if len(settings.ClientID) != 0 {
		clientConfig.ClientID = settings.ClientID
	}
	if len(settings.ClientSecret) != 0 {
		clientConfig.ClientSecret = settings.ClientSecret
	}
	if settings.SkipSSLVerify {
		clientConfig.SkipSslValidation = settings.SkipSSLVerify
	}
	if settings.TimeoutSeconds != 0 {
		clientConfig.HttpClient = &http.Client{
			Timeout: time.Duration(settings.TimeoutSeconds) * time.Second,
		}
	}
	return &PlatformClientConfiguration{
		Config:     clientConfig,
		createFunc: cfclient.NewClient,
	}
}
