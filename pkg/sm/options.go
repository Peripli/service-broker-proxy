/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sm

import (
	"crypto/tls"
	"time"

	"net/http"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/pkg/errors"
)

// Settings type holds SM Client config properties
type Settings struct {
	User                 string
	Password             string
	URL                  string
	OSBAPIPath           string        `mapstructure:"osb_api_path"`
	NotificationsAPIPath string        `mapstructure:"notifications_api_path"`
	RequestTimeout       time.Duration `mapstructure:"request_timeout"`
	SkipSSLValidation    bool          `mapstructure:"skip_ssl_validation"`
	VisibilitiesPageSize int           `mapstructure:"visibilities_page_size"`
	TLSClientKey         string        `mapstructure:"tls_client_id"`
	TLSClientCertificate string        `mapstructure:"tls_client_certificate"`
	Transport            http.RoundTripper
}

// DefaultSettings builds a default Service Manager Settings
func DefaultSettings() *Settings {
	return &Settings{
		User:                 "",
		Password:             "",
		URL:                  "",
		OSBAPIPath:           "/v1/osb",
		NotificationsAPIPath: "/v1/notifications",
		RequestTimeout:       60 * time.Second,
		SkipSSLValidation:    false,
		Transport:            nil,
	}
}

// NewSettings builds a Service Manager Settings from the provided Environment
func NewSettings(env env.Environment) (*Settings, error) {
	config := struct {
		Sm *Settings
	}{DefaultSettings()}

	if err := env.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling SM configuration")
	}

	return config.Sm, nil
}

// GetCertificates returns the client TLS certificates if configured
func (c *Settings) GetCertificates() ([]tls.Certificate, error) {
	var tlsCerts []tls.Certificate
	if len(c.TLSClientCertificate) != 0 || len(c.TLSClientKey) != 0 {
		cert, err := tls.X509KeyPair([]byte(c.TLSClientCertificate), []byte(c.TLSClientKey))
		if err != nil {
			return nil, errors.New("invalidate TLS configuration: " + err.Error())
		}

		tlsCerts = append(tlsCerts, cert)
	}

	return tlsCerts, nil
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Settings) Validate() error {
	if len(c.User) == 0 {
		return errors.New("SM configuration User missing")
	}
	if len(c.Password) == 0 {
		return errors.New("SM configuration Password missing")
	}
	if len(c.URL) == 0 {
		return errors.New("SM configuration URL missing")
	}
	if len(c.OSBAPIPath) == 0 {
		return errors.New("SM configuration OSB API Path missing")
	}
	if c.NotificationsAPIPath == "" {
		return errors.New("SM configuration Notifications API path must be non-empty string")
	}
	if c.RequestTimeout == 0 {
		return errors.New("SM configuration RequestTimeout missing")
	}
	return nil
}
