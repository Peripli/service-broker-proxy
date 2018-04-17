package cf

import (
	"fmt"
	"net/http"

	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/spf13/viper"
)

type RegistrationDetails struct {
	User     string
	Password string
}

func (rd RegistrationDetails) String() string {
	return fmt.Sprintf("User: %s Host: %s", rd.User)
}

type PlatformClientConfiguration struct {
	*cfclient.Config

	Reg        *RegistrationDetails
	createFunc func(*cfclient.Config) (*cfclient.Client, error)
}

var _ platform.ClientConfiguration = &PlatformClientConfiguration{}

func (c *PlatformClientConfiguration) CreateFunc() (platform.Client, error) {
	return NewClient(c)
}

func (c *PlatformClientConfiguration) Validate() error {
	if c.createFunc == nil {
		return fmt.Errorf("platform config error: createFunc missing")
	}
	if c.Config == nil {
		return fmt.Errorf("platform config error: CF config missing")
	}
	if len(c.Config.ApiAddress) == 0 {
		return fmt.Errorf("platform config error: CF ApiAddress missing")
	}
	if c.Reg == nil {
		return fmt.Errorf("platform config error: Registration credentials missing")
	}
	if len(c.Reg.User) == 0 {
		return fmt.Errorf("platform config error: Registration details user missing")
	}
	if len(c.Reg.Password) == 0 {
		return fmt.Errorf("platform config error: Registration details password missing")
	}
	return nil
}

type settings struct {
	Api            string
	ClientID       string
	ClientSecret   string
	Username       string
	Password       string
	SkipSSLVerify  bool
	TimeoutSeconds int
	Reg            *RegistrationDetails
}

func DefaultConfig() (*PlatformClientConfiguration, error) {
	platformSettings := &struct {
		Cf *settings
	}{
		Cf: &settings{},
	}
	//TODO BindEnv
	if err := viper.Unmarshal(platformSettings); err != nil {
		return nil, err
	}

	clientConfig := cfclient.DefaultConfig()

	if len(platformSettings.Cf.Api) != 0 {
		clientConfig.ApiAddress = platformSettings.Cf.Api
	}
	if len(platformSettings.Cf.ClientID) != 0 {
		clientConfig.ClientID = platformSettings.Cf.ClientID
	}
	if len(platformSettings.Cf.ClientSecret) != 0 {
		clientConfig.ClientSecret = platformSettings.Cf.ClientSecret
	}
	if len(platformSettings.Cf.Username) != 0 {
		clientConfig.ClientID = platformSettings.Cf.ClientID
	}
	if len(platformSettings.Cf.Password) != 0 {
		clientConfig.ClientSecret = platformSettings.Cf.ClientSecret
	}
	if platformSettings.Cf.SkipSSLVerify {
		clientConfig.SkipSslValidation = platformSettings.Cf.SkipSSLVerify
	}
	if platformSettings.Cf.TimeoutSeconds != 0 {
		clientConfig.HttpClient = &http.Client{
			Timeout: time.Duration(platformSettings.Cf.TimeoutSeconds) * time.Second,
		}
	}
	return &PlatformClientConfiguration{
		Config:     clientConfig,
		Reg:        platformSettings.Cf.Reg,
		createFunc: cfclient.NewClient,
	}, nil
}
