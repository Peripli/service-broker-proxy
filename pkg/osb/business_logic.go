package osb

import (
	"net/http"

	"github.com/pmorie/osb-broker-lib/pkg/broker"

	"errors"

	"fmt"

	"github.com/gorilla/mux"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
)

// BusinessLogic provides an implementation of the osb.BusinessLogic interface.
type BusinessLogic struct {
	createFunc      osbc.CreateFunc
	osbClientConfig *osbc.ClientConfiguration
}

var _ broker.Interface = &BusinessLogic{}

func NewBusinessLogic(config *ClientConfiguration) (*BusinessLogic, error) {
	return &BusinessLogic{
		osbClientConfig: config.ClientConfiguration,
		createFunc:      config.CreateFunc,
	}, nil
}

func (b *BusinessLogic) GetCatalog(c *broker.RequestContext) (*osbc.CatalogResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.GetCatalog()
}

func (b *BusinessLogic) Provision(request *osbc.ProvisionRequest, c *broker.RequestContext) (*osbc.ProvisionResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.ProvisionInstance(request)
}

func (b *BusinessLogic) Deprovision(request *osbc.DeprovisionRequest, c *broker.RequestContext) (*osbc.DeprovisionResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.DeprovisionInstance(request)
}

func (b *BusinessLogic) LastOperation(request *osbc.LastOperationRequest, c *broker.RequestContext) (*osbc.LastOperationResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.PollLastOperation(request)
}

func (b *BusinessLogic) Bind(request *osbc.BindRequest, c *broker.RequestContext) (*osbc.BindResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.Bind(request)
}

func (b *BusinessLogic) Unbind(request *osbc.UnbindRequest, c *broker.RequestContext) (*osbc.UnbindResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.Unbind(request)
}

func (b *BusinessLogic) Update(request *osbc.UpdateInstanceRequest, c *broker.RequestContext) (*osbc.UpdateInstanceResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	return client.UpdateInstance(request)
}

func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
	if version == "" {
		return errors.New("X-Broker-API-Version Header not set")
	}

	apiVersionHeader := b.osbClientConfig.APIVersion.HeaderValue()
	if apiVersionHeader != version {
		return errors.New("X-Broker-API-Version Header must be " + apiVersionHeader + " but was " + version)
	}
	return nil
}

func osbClient(request *http.Request, config osbc.ClientConfiguration, createFunc osbc.CreateFunc) (osbc.Client, error) {
	vars := mux.Vars(request)
	brokerID, ok := vars["brokerID"]
	if !ok {
		errMsg := fmt.Sprintf("brokerId path parameter missing from %s", request.Host)
		logrus.Error("Error building OSB client for proxy business logic: ", errMsg)
		return nil, osbc.HTTPStatusCodeError{
			StatusCode:  http.StatusBadRequest,
			Description: &errMsg,
		}
	}
	// assuming that we decide that osb api of SM is accessible with brokerID path parameter
	config.URL = config.URL + "/" + brokerID
	config.Name = config.Name + "-" + brokerID
	logrus.Debug("Building OSB client for broker with name: ", config.Name, " accesible at: ", config.URL)
	return createFunc(&config)
}
