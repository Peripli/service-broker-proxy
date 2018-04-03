package sbproxy

import (
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/task"
	"github.com/spf13/viper"
)

type ServerConfiguration struct {
	Port       int
	LogLevel   string
	LogFormat  string
	TimeoutSec int
	TLSKey     string
	TLSCert    string
	Reg        *task.RegistrationDetails
}

func (c *ServerConfiguration) Validate() error {
	if c.Port == 0 {
		return fmt.Errorf("Server Config error: Port missing")
	}
	if len(c.LogLevel) == 0 {
		return fmt.Errorf("Server Config error: LogLevel missing")
	}
	if len(c.LogFormat) == 0 {
		return fmt.Errorf("Server Config error: LogFormat missing")
	}
	if c.TimeoutSec == 0 {
		return fmt.Errorf("Server Config error: TimeoutSec missing")
	}
	if (c.TLSCert != "" || c.TLSKey != "") &&
		(c.TLSCert == "" || c.TLSKey == "") {
		return fmt.Errorf("Server Config error: To use TLS both TLSCert and TLSKey must be provided")
	}
	if c.Reg == nil {
		return fmt.Errorf("Server Config error: Registration Details missing")
	}

	if len(c.Reg.User) == 0 {
		return fmt.Errorf("Server Config error: Registration Details User missing")
	}
	if len(c.Reg.Password) == 0 {
		return fmt.Errorf("Server Config error: Registration Details Password missing")
	}
	if len(c.Reg.Host) == 0 {
		return fmt.Errorf("Server Config error: Registration Details Host missing")
	}
	return nil
}

func DefaultConfig() *ServerConfiguration {
	serverConfig := &ServerConfiguration{
		Port:       8080,
		LogLevel:   "debug",
		LogFormat:  "text",
		TimeoutSec: 15,
		TLSKey:     "",
		TLSCert:    "",
		Reg: &task.RegistrationDetails{
			User:     "admin",
			Password: "admin",
			Host:     "http://localhost:8080/proxy",
		},
	}
	viperServer := viper.Sub("server")
	viperServer.SetEnvPrefix("server")
	viperServer.Unmarshal(serverConfig)

	return serverConfig
}
