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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Peripli/service-manager/pkg/types"

	"time"

	"context"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/pkg/errors"
)

const (
	// APIInternalBrokers is the SM API for obtaining the brokers for this proxy
	APIInternalBrokers = "%s" + web.ServiceBrokersURL

	// APIVisibilities is the SM API for obtaining plan visibilities
	APIVisibilities = "%s" + web.VisibilitiesURL

	// APIPlans is the SM API for obtaining service plans
	APIPlans = "%s" + web.ServicePlansURL

	// APIServiceOfferings is the SM API for obtaining service offerings
	APIServiceOfferings = "%s" + web.ServiceOfferingsURL

	// APICredentials is the SM API for managing broker platform credentials
	APICredentials = "%s" + web.BrokerPlatformCredentialsURL
)

// ErrConflictingBrokerPlatformCredentials error returned from SM when broker platform credentials already exist
var ErrConflictingBrokerPlatformCredentials = errors.New("conflicting broker platform credentials")

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error)
	GetVisibilities(ctx context.Context) ([]*types.Visibility, error)
	GetPlans(ctx context.Context) ([]*types.ServicePlan, error)
	GetServiceOfferings(ctx context.Context) ([]*types.ServiceOffering, error)
	PutCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) error
}

// ServiceManagerClient allows consuming Service Manager APIs
type ServiceManagerClient struct {
	host       string
	httpClient *http.Client
}

// NewClient builds a new Service Manager Client from the provided configuration
func NewClient(config *Settings) (*ServiceManagerClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	httpClient.Timeout = time.Duration(config.RequestTimeout)
	tr := config.Transport

	if tr == nil {
		tr = &SkipSSLTransport{
			SkipSslValidation: config.SkipSSLValidation,
		}
	}

	httpClient.Transport = &BasicAuthTransport{
		Username: config.User,
		Password: config.Password,
		Rt:       tr,
	}

	return &ServiceManagerClient{
		host:       config.URL,
		httpClient: httpClient,
	}, nil
}

// GetBrokers calls the Service Manager in order to obtain all brokers that need to be registered
// in the service broker proxy
func (c *ServiceManagerClient) GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error) {
	log.C(ctx).Debugf("Getting brokers for proxy from Service Manager at %s", c.host)

	result := make([]*types.ServiceBroker, 0)
	err := c.call(ctx, fmt.Sprintf(APIInternalBrokers, c.host), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	return result, nil
}

// GetVisibilities returns plan visibilities from Service Manager
func (c *ServiceManagerClient) GetVisibilities(ctx context.Context) ([]*types.Visibility, error) {
	log.C(ctx).Debugf("Getting visibilities for proxy from Service Manager at %s", c.host)

	result := make([]*types.Visibility, 0)
	err := c.call(ctx, fmt.Sprintf(APIVisibilities, c.host), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting visibilities from Service Manager")
	}

	return result, nil
}

// GetPlans returns plans from Service Manager
func (c *ServiceManagerClient) GetPlans(ctx context.Context) ([]*types.ServicePlan, error) {
	log.C(ctx).Debugf("Getting service plans for proxy from Service Manager at %s", c.host)

	var result []*types.ServicePlan
	err := c.call(ctx, fmt.Sprintf(APIPlans, c.host), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service plans from Service Manager")
	}

	return result, nil
}

// GetServiceOfferings returns service offerings from Service Manager
func (c *ServiceManagerClient) GetServiceOfferings(ctx context.Context) ([]*types.ServiceOffering, error) {
	log.C(ctx).Debugf("Getting service offerings from Service Manager at %s", c.host)

	var result []*types.ServiceOffering
	err := c.call(ctx, fmt.Sprintf(APIServiceOfferings, c.host), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service offerings from Service Manager")
	}

	return result, nil
}

//PutCredentials sends new broker platform credentials to Service Manager
func (c *ServiceManagerClient) PutCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) error {
	log.C(ctx).Debugf("Putting credentials in Service Manager at %s", c.host)

	body, err := json.Marshal(credentials)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf(APICredentials, c.host), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "error registering credentials in Service Manager")
	}

	switch response.StatusCode {
	case http.StatusOK:
		log.C(ctx).Debugf("Successfully putting credentials in Service Manager at: %s", c.host)
	case http.StatusConflict:
		log.C(ctx).Debugf("Credentials could not be persisted. Existing credentials were found in Service Manager at: %s", c.host)
		return ErrConflictingBrokerPlatformCredentials
	default:
		return fmt.Errorf("unexpected response status code received (%v) upon credentials registration", response.StatusCode)
	}

	return nil
}

func (c *ServiceManagerClient) call(ctx context.Context, smURL string, params map[string]string, list interface{}) error {
	fullURL, err := url.Parse(smURL)
	if err != nil {
		return err
	}
	q := fullURL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	fullURL.RawQuery = q.Encode()
	return util.ListAll(ctx, c.httpClient.Do, fullURL.String(), list)
}
