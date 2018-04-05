package task

import (
	"sync"

	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/sirupsen/logrus"
)

type RegistrationDetails struct {
	User     string
	Password string
	Host     string
}

func (rd RegistrationDetails) String() string {
	return fmt.Sprintf("User: %s Host: %s", rd.User, rd.Host)
}

type SBProxyRegistration struct {
	group          sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	details        *RegistrationDetails
}

func New(group sync.WaitGroup, platoformClient platform.Client, smClient sm.Client, details *RegistrationDetails) *SBProxyRegistration {
	return &SBProxyRegistration{
		group:          group,
		platformClient: platoformClient,
		smClient:       smClient,
		details:        details,
	}
}

func (r SBProxyRegistration) Run() {
	r.group.Add(1)
	defer r.group.Done()
	r.run()
}

func (r SBProxyRegistration) run() {
	logrus.Debug("Running broker registration task with details: ", r.details)
	registeredBrokers, err := r.platformClient.GetBrokers()
	if err != nil {
		logrus.Error("An error occurred while obtaining already registered brokers: ", err)
		return
	}

	proxyBrokers, err := r.smClient.GetBrokers()
	if err != nil {
		logrus.Error("An error occurred while obtaining brokers that have to be registered at the platform: ", err)
		return
	}

	updateBrokerRegistrations(r.createBrokerRegistration, proxyBrokers.ServiceBrokers, registeredBrokers.ServiceBrokers)
	updateBrokerRegistrations(r.deleteBrokerRegistration, registeredBrokers.ServiceBrokers, proxyBrokers.ServiceBrokers)
}

func (r SBProxyRegistration) deleteBrokerRegistration(broker *platform.ServiceBroker) {
	deleteRequest := &platform.DeleteServiceBrokerRequest{
		Guid: broker.Guid,
	}
	if err := r.platformClient.DeleteBroker(deleteRequest); err != nil {
		logrus.Error("Error during broker deletion: ", err)
	}
}

func (r SBProxyRegistration) createBrokerRegistration(broker *platform.ServiceBroker) {
	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      "sm-proxy-" + broker.Name,
		Username:  r.details.User,
		Password:  r.details.Password,
		BrokerURL: r.details.Host + "/" + broker.Guid,
	}
	if _, err := r.platformClient.CreateBroker(createRequest); err != nil {
		logrus.Error("Error during broker registration: ", err)
	}
}

func updateBrokerRegistrations(updateOp func(broker *platform.ServiceBroker), a, b []platform.ServiceBroker) {
	mb := make(map[string]platform.ServiceBroker)
	for _, broker := range b {
		mb[broker.Guid] = broker
	}
	for _, broker := range a {
		if _, ok := mb[broker.Guid]; !ok {
			updateOp(&broker)
		}
	}
}
