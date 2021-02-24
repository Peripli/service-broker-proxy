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
	"strings"

	"github.com/pkg/errors"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

// ErrConflictingBrokerPlatformCredentials error returned from SM when broker platform credentials already exist
var ErrConflictingBrokerPlatformCredentials = errors.New("conflicting broker platform credentials")

// ErrBrokerPlatformCredentialsNotFound error returned from SM when the given broker platform credentials id does not exist
var ErrBrokerPlatformCredentialsNotFound = errors.New("credentials not found")

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error)
	GetVisibilities(ctx context.Context, planIDs []string) ([]*types.Visibility, error)
	GetPlans(ctx context.Context) ([]*types.ServicePlan, error)
	GetServiceOfferings(ctx context.Context) ([]*types.ServiceOffering, error)
	PutCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) (*types.BrokerPlatformCredential, error)
	ActivateCredentials(ctx context.Context, credentialsID string) error
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
	httpClient.Timeout = config.RequestTimeout
	tlsCerts, err := config.GetCertificates()
	if err != nil {
		return nil, err
	}

	tr := config.Transport

	if tr == nil {
		tr = &SSLTransport{
			SkipSslValidation: config.SkipSSLValidation,
			TLSCertificates:   tlsCerts,
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
func (c *ServiceManagerClient) GetVisibilities(ctx context.Context, planIDs []string) ([]*types.Visibility, error) {
	log.C(ctx).Debugf("Getting visibilities for proxy from Service Manager at %s", c.url)
	if planIDs == nil || len(planIDs) == 0 {
		return nil, fmt.Errorf("error getting visibilities from Service Manager. Plan IDs must be provided")
	}
	plansStr := "('" + strings.Join(planIDs, "','") + "')"
	params := map[string]string{
		"fieldQuery": "ready eq true and service_plan_id in " + plansStr,
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

//PutCredentials sends new broker platform credentials to Service Manager
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
		log.C(ctx).Debugf("Successfully putting credentials in Service Manager at: %s", c.url)
	case http.StatusConflict:
		log.C(ctx).Debugf("Credentials could not be persisted. Existing credentials were found in Service Manager at: %s", c.url)
		return nil, ErrConflictingBrokerPlatformCredentials
	default:
		return nil, fmt.Errorf("unexpected response status code received (%v) upon credentials registration", response.StatusCode)
	}
	var brokerPlatformCred = types.BrokerPlatformCredential{}
	if err = util.BodyToObject(response.Body, &brokerPlatformCred); err != nil {
		log.C(ctx).WithError(err).Error("error reading response body")
		return nil, err
	}

	return &brokerPlatformCred, nil
}

//ActivateCredentials notifies Service Manager that the new broker credentials are valid and verified
func (c *ServiceManagerClient) ActivateCredentials(ctx context.Context, credentialsID string) error {
	log.C(ctx).Debugf("Activating credentials in Service Manager at %s", c.url)

	req, err := http.NewRequest(http.MethodPut, c.getURL(fmt.Sprintf("%s/%s/activate", web.BrokerPlatformCredentialsURL, credentialsID)), bytes.NewBufferString("{}"))
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
		log.C(ctx).Debugf("Successfully activating credentials in Service Manager at: %s", c.url)
	case http.StatusNotFound:
		log.C(ctx).Debugf("Credentials could not be activated. credentials with id %s were not found in Service Manager at: %s", credentialsID, c.url)
		return ErrBrokerPlatformCredentialsNotFound
	default:
		return fmt.Errorf("unexpected response status code received (%v) upon credentials activation", response.StatusCode)
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
