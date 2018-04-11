package sm

import (
	"fmt"

	"github.com/spf13/viper"
)

type ClientConfiguration struct {
	User           string
	Password       string
	Host           string
	TimeoutSeconds int
	CreateFunc     func(config *ClientConfiguration) (Client, error)
}

func (c *ClientConfiguration) Validate() error {
	if len(c.User) == 0 {
		return fmt.Errorf("SM Config error: User missing")
	}
	if len(c.Password) == 0 {
		return fmt.Errorf("SM Config error: Password missing")
	}
	if len(c.Host) == 0 {
		return fmt.Errorf("SM Config error: Host missing")
	}
	if c.TimeoutSeconds == 0 {
		return fmt.Errorf("SM Config error: TimeoutSeconds missing")
	}
	if c.CreateFunc == nil {
		return fmt.Errorf("SM Config error: CreateFunc missing")
	}
	return nil
}

//TODO https://github.com/spf13/viper/issues/239
func DefaultConfig() (*ClientConfiguration, error) {
	config := &ClientConfiguration{
		User:           "admin",
		Password:       "admin",
		Host:           "http://localhost:8080/sm",
		TimeoutSeconds: 10,
		CreateFunc:     NewClient,
	}
	smConfig := &struct {
		Sm *ClientConfiguration
	}{
		Sm: config,
		}
		//TODO BindEnv
	if err := viper.Unmarshal(smConfig); err != nil {
		return nil, err
	}

	return config, nil
}
