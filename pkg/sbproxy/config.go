package sbproxy

import (
	"fmt"

	"github.com/spf13/viper"
)

type ServerConfiguration struct {
	Port       int
	LogLevel   string
	LogFormat  string
	TimeoutSec int
	TLSKey     string
	TLSCert    string
	Host       string
}

func (c *ServerConfiguration) Validate() error {
	if c.Port == 0 {
		return fmt.Errorf("server Config error: Port missing")
	}
	if len(c.LogLevel) == 0 {
		return fmt.Errorf("server Config error: LogLevel missing")
	}
	if len(c.LogFormat) == 0 {
		return fmt.Errorf("server Config error: LogFormat missing")
	}
	if c.TimeoutSec == 0 {
		return fmt.Errorf("server Config error: TimeoutSec missing")
	}
	if (c.TLSCert != "" || c.TLSKey != "") &&
		(c.TLSCert == "" || c.TLSKey == "") {
		return fmt.Errorf("server Config error: To use TLS both TLSCert and TLSKey must be provided")
	}

	if len(c.Host) == 0 {
		return fmt.Errorf("server Config error: Host missing")
	}
	return nil
}

func DefaultConfig() (*ServerConfiguration, error) {
	config := &ServerConfiguration{
		Port:       8080,
		LogLevel:   "debug",
		LogFormat:  "text",
		TimeoutSec: 15,
		TLSKey:     "",
		TLSCert:    "",
		Host:       "http://localhost:8080/proxy",
	}
	serverConfig := &struct {
		Server *ServerConfiguration
	}{
		Server: config,
	}
	//TODO BindEnv
	if err := viper.Unmarshal(serverConfig); err != nil {
		return nil, err
	}
	return config, nil
}
