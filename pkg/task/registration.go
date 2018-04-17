package task

import (
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/sirupsen/logrus"
)

type SBProxyRegistration struct {
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	smHost         string
}

func New(group *sync.WaitGroup, platoformClient platform.Client, smClient sm.Client, smHost string) *SBProxyRegistration {
	return &SBProxyRegistration{
		group:          group,
		platformClient: platoformClient,
		smClient:       smClient,
		smHost:         smHost,
	}
}

func (r SBProxyRegistration) Run() {
	r.group.Add(1)
	defer r.group.Done()
	r.run()
}

func (r SBProxyRegistration) run() {
	logrus.Debug("Running broker registration task...")
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

	updateBrokerRegistrations(r.createBrokerRegistration, proxyBrokers, registeredBrokers)
	updateBrokerRegistrations(r.deleteBrokerRegistration, registeredBrokers, proxyBrokers)
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
		BrokerURL: r.smHost + "/" + broker.Guid,
		SpaceGUID: broker.Guid,
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
