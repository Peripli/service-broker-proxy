package sm

import (
	"fmt"
	"net/http"

	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// APIInternalBrokers is the SM API for obtaining the brokers for this proxy
const APIInternalBrokers = "%s/v1/service_brokers"

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers() ([]platform.ServiceBroker, error)
}

type serviceManagerClient struct {
	Config     *Config
	httpClient *http.Client
}

var _ Client = &serviceManagerClient{}

// NewClient builds a new Service Manager Client from the provided configuration
func NewClient(config *Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: time.Duration(config.RequestTimeout) * time.Second,
	}

	if config.User != "" && config.Password != "" {
		httpClient.Transport = BasicAuthTransport{
			username: config.User,
			password: config.Password,
			rt:       http.DefaultTransport,
		}
	}

	client := &serviceManagerClient{
		Config:     config,
		httpClient: httpClient,
	}

	return client, nil
}

// GetBrokers calls the Service Manager in order to obtain all brokers that need to be registered
// in the service broker proxy
func (c *serviceManagerClient) GetBrokers() ([]platform.ServiceBroker, error) {
	logrus.Debugf("Getting brokers for proxy from Service Manager at %s", c.Config.Host)
	URL := fmt.Sprintf(APIInternalBrokers, c.Config.Host)
	response, err := util.SendClientRequest(c.httpClient, http.MethodGet, URL, map[string]string{"catalog": "true"}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	list := &Brokers{}
	switch response.StatusCode {
	case http.StatusOK:
		if err = util.ReadClientResponseContent(list, response.Body); err != nil {
			return nil, errors.Wrapf(err, "error getting content from body of response with status %s", response.Status)
		}
	default:
		return nil, errors.WithStack(util.HandleClientResponseError(response))
	}

	return c.packResponse(list), nil
}

func (c *serviceManagerClient) packResponse(list *Brokers) []platform.ServiceBroker {
	brokers := make([]platform.ServiceBroker, 0, len(list.Brokers))
	for _, broker := range list.Brokers {
		b := platform.ServiceBroker{
			GUID:      broker.ID,
			Name:      broker.Name,
			BrokerURL: broker.BrokerURL,
			Catalog:   broker.Catalog,
			Metadata:  broker.Metadata,
		}
		brokers = append(brokers, b)
	}
	return brokers
}
