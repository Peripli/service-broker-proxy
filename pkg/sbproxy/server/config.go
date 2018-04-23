package server

import (
	"github.com/pkg/errors"

	"github.com/spf13/viper"
)

type AppConfiguration struct {
	Port       int
	LogLevel   string
	LogFormat  string
	TimeoutSec int
	TLSKey     string
	TLSCert    string
	Host       string
}

func (c *AppConfiguration) Validate() error {
	if c.Port == 0 {
		return errors.New("application configuration Port missing")
	}
	if len(c.LogLevel) == 0 {
		return errors.New("application configuration LogLevel missing")
	}
	if len(c.LogFormat) == 0 {
		return errors.New("application configuration LogFormat missing")
	}
	if c.TimeoutSec == 0 {
		return errors.New("application configuration TimeoutSec missing")
	}
	if (c.TLSCert != "" || c.TLSKey != "") &&
		(c.TLSCert == "" || c.TLSKey == "") {
		return errors.New("application configuration both TLSCert and TLSKey must be provided to use TLS")
	}

	if len(c.Host) == 0 {
		return errors.New("application configuration Host missing")
	}
	return nil
}

func DefaultConfig() (*AppConfiguration, error) {
	config := &AppConfiguration{
		Port:       8080,
		LogLevel:   "debug",
		LogFormat:  "text",
		TimeoutSec: 15,
		TLSKey:     "",
		TLSCert:    "",
		Host:       "",
	}
	appConfig := &struct {
		App *AppConfiguration
	}{
		App: config,
	}
	//TODO BindEnv
	if err := viper.Unmarshal(appConfig); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling app configuration")
	}
	return config, nil
}
