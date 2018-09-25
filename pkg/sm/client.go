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
	GetBrokers(ctx context.Context) ([]platform.ServiceBroker, error)
}

// ServiceManagerClient allows consuming APIs from a Service Manager
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
		host:       config.Host,
		httpClient: httpClient,
	}, nil
}

// GetBrokers calls the Service Manager in order to obtain all brokers t	hat need to be registered
// in the service broker proxy
func (c *ServiceManagerClient) GetBrokers(ctx context.Context) ([]Broker, error) {
	log.C(ctx).Debugf("Getting brokers for proxy from Service Manager at %s", c.host)
	URL := fmt.Sprintf(APIInternalBrokers, c.host)
	response, err := util.SendRequest(ctx, c.httpClient.Do, http.MethodGet, URL, map[string]string{"catalog": "true"}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	list := &Brokers{}
	switch response.StatusCode {
	case http.StatusOK:
		if err = util.BodyToObject(response.Body, list); err != nil {
			return nil, errors.Wrapf(err, "error getting content from body of response with status %s", response.Status)
		}
	default:
		return nil, errors.WithStack(util.HandleResponseError(response))
	}

	return list.Brokers, nil
}
