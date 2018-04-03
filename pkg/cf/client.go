package cf

import (
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/sirupsen/logrus"
)

//TODO This package should be in seperate repository (CF Wrapper Module) for the proxy
type PlatformClient struct {
	cfClient *cfclient.Client
}

var _ platform.Client = &PlatformClient{}

func NewClient(config *PlatformClientConfiguration) (platform.Client, error) {
	cfClient, err := config.createFunc(config.Config)
	if err != nil {
		return nil, err
	}
	return &PlatformClient{cfClient: cfClient}, nil
}

func (b PlatformClient) GetBrokers() (*platform.ServiceBrokerList, error) {
	logrus.Debug("Getting platform brokers...")
	logrus.Debug("Obtaining CF API access token...")
	_, err := b.cfClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("Error obtaining CF access token: %s", err)
	}
	logrus.Debug("Listing brokers via CF client...")
	brokers, err := b.cfClient.ListServiceBrokers()
	if err != nil {
		return nil, fmt.Errorf("Error listing service brokers via CF client: %s", err)
	}
	var clientBrokers []platform.ServiceBroker
	for _, broker := range brokers {
		serviceBroker := platform.ServiceBroker{Guid: broker.Guid,
			Name:      broker.Name,
			Username:  broker.Username,
			Password:  broker.Password,
			BrokerURL: broker.BrokerURL,
		}
		clientBrokers = append(clientBrokers, serviceBroker)
	}
	return &platform.ServiceBrokerList{ServiceBrokers: clientBrokers}, nil
}

func (b PlatformClient) CreateBrokers(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Creating broker with name [%s]...", r.Name)
	logrus.Debug("Obtaining CF API access token...")
	_, err := b.cfClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("Error obtaining CF access token: %s", err)
	}

	request := cfclient.CreateServiceBrokerRequest{
		Name:      r.Name,
		Username:  r.Username,
		Password:  r.Password,
		BrokerURL: r.BrokerURL,
		SpaceGUID: r.SpaceGUID,
	}

	logrus.Debugf("Creating broker with name [%s] via CF client", request.Name)
	broker, err := b.cfClient.CreateServiceBroker(request)
	if err != nil {
		return nil, fmt.Errorf("Error creating broker: %s", err)
	}

	response := &platform.ServiceBroker{
		Guid:      broker.Guid,
		Name:      broker.Name,
		Username:  broker.Username,
		Password:  broker.Password,
		BrokerURL: broker.BrokerURL,
	}

	return response, nil
}

func (b PlatformClient) DeleteBrokers(r *platform.DeleteServiceBrokerRequest) error {
	logrus.Debugf("Deleting broker with GUID [%s]...", r.Guid)
	logrus.Debug("Obtaining CF API access token...")
	_, err := b.cfClient.GetToken()
	if err != nil {
		return fmt.Errorf("Error obtaining CF access token: %s", err)
	}
	logrus.Debugf("Deleting broker with guid [%s] via CF client", r.Guid)
	if err = b.cfClient.DeleteServiceBroker(r.Guid); err != nil {
		return fmt.Errorf("Error deleting broker: %s", err)
	}
	return nil
}

func (b PlatformClient) UpdateBrokers(r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Updating broker with name [%s]...", r.Name)
	logrus.Debug("Obtaining CF API access token...")
	_, err := b.cfClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("Error obtaining CF access token: %s", err)
	}
	request := cfclient.UpdateServiceBrokerRequest{
		Name:      r.Name,
		Username:  r.Username,
		Password:  r.Password,
		BrokerURL: r.BrokerURL,
	}

	logrus.Debugf("Updating broker with GUID [%s]", r.Guid)
	broker, err := b.cfClient.UpdateServiceBroker(r.Guid, request)
	if err != nil {
		return nil, fmt.Errorf("Error updating broker: %s", err)
	}
	response := &platform.ServiceBroker{
		Guid:      broker.Guid,
		Name:      broker.Name,
		Username:  broker.Username,
		Password:  broker.Password,
		BrokerURL: broker.BrokerURL,
	}
	return response, nil
}
