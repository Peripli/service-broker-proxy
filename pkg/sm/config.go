package sm

import (
	"fmt"

	"github.com/spf13/viper"
)

type ClientConfiguration struct {
	User           string
	Password       string
	Uri            string
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
	if len(c.Uri) == 0 {
		return fmt.Errorf("SM Config error: Uri missing")
	}
	if c.TimeoutSeconds == 0 {
		return fmt.Errorf("SM Config error: TimeoutSeconds missing")
	}
	if c.CreateFunc == nil {
		return fmt.Errorf("SM Config error: CreateFunc missing")
	}
	return nil
}

func DefaultConfig() *ClientConfiguration {
	config := &ClientConfiguration{
		User:           "admin",
		Password:       "admin",
		Uri:            "http://localhost:8080/sm",
		TimeoutSeconds: 10,
		CreateFunc:     NewClient,
	}
	viperServer := viper.Sub("sm")
	viperServer.SetEnvPrefix("sm")
	viperServer.Unmarshal(config)
	return config
}
