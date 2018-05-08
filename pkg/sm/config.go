package sm

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/pkg/errors"
)

// ClientConfiguration type holds SM Client config properties
type ClientConfiguration struct {
	User           string
	Password       string
	Host           string
	OsbApi         string
	TimeoutSeconds int
	CreateFunc     func(config *ClientConfiguration) (Client, error)
}

// Settings type is used to unmarshal SM ClientConfiguration from the environment
type Settings struct {
	Sm *ClientConfiguration
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfiguration) Validate() error {
	if len(c.User) == 0 {
		return errors.New("SM configuration User missing")
	}
	if len(c.Password) == 0 {
		return errors.New("SM configuration Password missing")
	}
	if len(c.Host) == 0 {
		return errors.New("SM configuration Host missing")
	}
	if len(c.OsbApi) == 0 {
		return errors.New("SM configuration OSB API missing")
	}
	if c.TimeoutSeconds == 0 {
		return errors.New("SM configuration TimeoutSeconds missing")
	}
	if c.CreateFunc == nil {
		return errors.New("SM configuration CreateFunc missing")
	}
	return nil
}

// DefaultConfig builds a default Service Manager ClientConfiguration
func DefaultConfig() *ClientConfiguration {
	return &ClientConfiguration{
		User:           "admin",
		Password:       "admin",
		Host:           "",
		TimeoutSeconds: 10,
		CreateFunc:     NewClient,
	}
}

// NewConfig builds a Service Manager ClientConfiguration from the provided Environment
func NewConfig(env env.Environment) (*ClientConfiguration, error) {
	config := DefaultConfig()

	smConfig := &Settings{
		Sm: config,
	}
	if err := env.Unmarshal(smConfig); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling SM configuration")
	}

	return config, nil
}
