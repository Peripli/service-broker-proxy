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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

// ErrConflictingBrokerPlatformCredentials error returned from SM when broker platform credentials already exist
var ErrConflictingBrokerPlatformCredentials = errors.New("conflicting broker platform credentials")

// ErrBrokerPlatformCredentialsNotFound error returned from SM when broker platform credentials already exist
var ErrBrokerPlatformCredentialsNotFound = errors.New("broker platform credentials not found")

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error)
	GetVisibilities(ctx context.Context) ([]*types.Visibility, error)
	GetPlans(ctx context.Context) ([]*types.ServicePlan, error)
	GetServiceOfferings(ctx context.Context) ([]*types.ServiceOffering, error)
	PutCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) (*types.BrokerPlatformCredential, error)
	RevertCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) error
}

// ServiceManagerClient allows consuming Service Manager APIs
type ServiceManagerClient struct {
	url                  string
	visibilitiesPageSize int

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
		url:                  config.URL,
		visibilitiesPageSize: config.VisibilitiesPageSize,
		httpClient:           httpClient,
	}, nil
}

// GetBrokers calls the Service Manager in order to obtain all brokers that need to be registered
// in the service broker proxy
func (c *ServiceManagerClient) GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error) {
	log.C(ctx).Debugf("Getting brokers for proxy from Service Manager at %s", c.url)

	result := make([]*types.ServiceBroker, 0)
	err := c.listAll(ctx, c.getURL(web.ServiceBrokersURL), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	return result, nil
}

// GetVisibilities returns plan visibilities from Service Manager
func (c *ServiceManagerClient) GetVisibilities(ctx context.Context) ([]*types.Visibility, error) {
	log.C(ctx).Debugf("Getting visibilities for proxy from Service Manager at %s", c.url)

	params := map[string]string{
		"fieldQuery": "ready eq true",
	}
	if c.visibilitiesPageSize > 0 {
		params["max_items"] = strconv.Itoa(c.visibilitiesPageSize)
	}
	result := make([]*types.Visibility, 0)
	err := c.listAll(ctx, c.getURL(web.VisibilitiesURL), params, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting visibilities from Service Manager")
	}

	return result, nil
}

// GetPlans returns plans from Service Manager
func (c *ServiceManagerClient) GetPlans(ctx context.Context) ([]*types.ServicePlan, error) {
	log.C(ctx).Debugf("Getting service plans for proxy from Service Manager at %s", c.url)

	var result []*types.ServicePlan
	err := c.listAll(ctx, c.getURL(web.ServicePlansURL), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service plans from Service Manager")
	}

	return result, nil
}

// GetServiceOfferings returns service offerings from Service Manager
func (c *ServiceManagerClient) GetServiceOfferings(ctx context.Context) ([]*types.ServiceOffering, error) {
	log.C(ctx).Debugf("Getting service offerings from Service Manager at %s", c.url)

	var result []*types.ServiceOffering
	err := c.listAll(ctx, c.getURL(web.ServiceOfferingsURL), map[string]string{
		"fieldQuery": "ready eq true",
	}, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service offerings from Service Manager")
	}

	return result, nil
}

// PutCredentials sends new broker platform credentials to Service Manager
func (c *ServiceManagerClient) PutCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) (*types.BrokerPlatformCredential, error) {
	log.C(ctx).Debugf("Putting credentials in Service Manager at %s", c.url)

	body, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPut, c.getURL(web.BrokerPlatformCredentialsURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error registering credentials in Service Manager")
	}

	switch response.StatusCode {
	case http.StatusOK:
		log.C(ctx).Infof("Successfully putting credentials in Service Manager at: %s", c.url)
	case http.StatusConflict:
		log.C(ctx).Errorf("Credentials could not be persisted. Existing credentials were found in Service Manager at: %s", c.url)
		return nil, ErrConflictingBrokerPlatformCredentials
	default:
		return nil, fmt.Errorf("unexpected response status code received (%v) upon credentials registration", response.StatusCode)
	}

	if err := util.BodyToObject(response.Body, credentials); err != nil {
		return nil, fmt.Errorf("Could not read object from body: %s", err)
	}

	return credentials, nil
}

// RevertCredentials move the old credentials as new. Intended to be used if the update of the broker in platform fails
func (c *ServiceManagerClient) RevertCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) error {
	endpoint := fmt.Sprintf("%s/%s", c.getURL(web.BrokerPlatformCredentialsURL), credentials.ID)
	log.C(ctx).Infof("Reverting credentials with id %s in Service Manager at %s", credentials.ID, endpoint)

	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "error reverting credentials in Service Manager")
	}

	switch response.StatusCode {
	case http.StatusOK:
		log.C(ctx).Infof("Successfully reverted credentials in Service Manager with id: %s", credentials.ID)
	case http.StatusNotFound:
		log.C(ctx).Errorf("Credentials with id %s could not be found", credentials.ID)
		return ErrBrokerPlatformCredentialsNotFound
	default:
		return fmt.Errorf("unexpected response status code received (%v) upon credentials reverting", response.StatusCode)
	}

	return nil
}

func (c *ServiceManagerClient) listAll(ctx context.Context, smURL string, params map[string]string, list interface{}) error {
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

func (c *ServiceManagerClient) getURL(route string) string {
	return c.url + route
}
