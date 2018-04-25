package task

import (
	"sync"

	"strings"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/sirupsen/logrus"
)

const ProxyBrokerPrefix = "sm-proxy-"

//TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type SBProxyRegistration struct {
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	proxyHost      string
}

type serviceBrokerReg struct {
	platform.ServiceBroker
	SmID string
}

func New(group *sync.WaitGroup, platformClient platform.Client, smClient sm.Client, proxyHost string) *SBProxyRegistration {
	return &SBProxyRegistration{
		group:          group,
		platformClient: platformClient,
		smClient:       smClient,
		proxyHost:      proxyHost,
	}
}

func (r SBProxyRegistration) Run() {
	logrus.Debug("STARTING broker registration task...")

	r.group.Add(1)
	defer r.group.Done()
	r.run()

	logrus.Debug("FINISHED broker registration task...")
}

func (r SBProxyRegistration) run() {

	// get all the registered proxy brokers from the platform
	brokersFromPlatform := r.getBrokersFromPlatform()

	// get all the brokers that are in SM and for which a proxy broker should be present in the platform
	brokersFromSM := r.getBrokersFromSM()

	// register brokers that are present in SM and missing from platform
	updateBrokerRegistrations(r.createBrokerRegistration, brokersFromSM, brokersFromPlatform)

	// unregister proxy brokers that are no longer in SM but are still in platform
	unregisteredBrokers := updateBrokerRegistrations(r.deleteBrokerRegistration, brokersFromPlatform, brokersFromSM)

	// trigger a fetch of catalogs of proxy brokers that are registered at platform
	updateBrokerRegistrations(r.fetchBrokerCatalogs, brokersFromPlatform, unregisteredBrokers)

}

func (r SBProxyRegistration) getBrokersFromPlatform() []serviceBrokerReg {

	registeredBrokers, err := r.platformClient.GetBrokers()
	if err != nil {
		logrus.WithError(err).Error("An error occurred while obtaining already registered brokers")
		return []serviceBrokerReg{}
	}

	brokersFromPlatform := make([]serviceBrokerReg, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		if !r.isBrokerProxy(broker) {
			logrus.WithFields(logFields(&broker)).Debug("Registration task SKIPPING registered broker as is not recognized to be proxy broker...")
			continue
		}

		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:],
		}
		brokersFromPlatform = append(brokersFromPlatform, brokerReg)
	}
	return brokersFromPlatform
}

func (r SBProxyRegistration) getBrokersFromSM() []serviceBrokerReg {
	proxyBrokers, err := r.smClient.GetBrokers()
	if err != nil {
		logrus.WithError(err).Error("An error occurred while obtaining brokers that have to be registered at the platform")
		return []serviceBrokerReg{}
	}

	brokersFromSM := make([]serviceBrokerReg, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.Guid,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}

	return brokersFromSM
}

func (r SBProxyRegistration) fetchBrokerCatalogs(broker platform.ServiceBroker) {
	if f, isFetcher := r.platformClient.(platform.Fetcher); isFetcher {
		if err := f.Fetch(&broker); err != nil {
			logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during fetching catalog...")
		}
	}
}

func (r SBProxyRegistration) createBrokerRegistration(broker platform.ServiceBroker) {
	logrus.WithFields(logFields(&broker)).Info("Registration task will attempt to create broker...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.Guid,
		BrokerURL: r.proxyHost + "/" + broker.Guid,
	}

	if _, err := r.platformClient.CreateBroker(createRequest); err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker creation")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(&broker)).Info("Registration task broker creation successful")
	}
}

func (r SBProxyRegistration) deleteBrokerRegistration(broker platform.ServiceBroker) {
	logrus.WithFields(logFields(&broker)).Info("Registration task will attempt to delete broker...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		Guid: broker.Guid,
		Name: broker.Name,
	}

	if err := r.platformClient.DeleteBroker(deleteRequest); err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker deletion")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(&broker)).Info("Registration task broker deletion successful")

	}
}

func (r SBProxyRegistration) isBrokerProxy(broker platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyHost) || strings.HasPrefix(broker.BrokerURL, "http://host.pcfdev.io:8083")
}

func updateBrokerRegistrations(updateOp func(broker platform.ServiceBroker), a, b []serviceBrokerReg) []serviceBrokerReg {
	affectedBrokers := make([]serviceBrokerReg, 0, 0)

	mb := make(map[string]serviceBrokerReg)
	for _, broker := range b {
		mb[broker.SmID] = broker
	}
	for _, broker := range a {
		if _, ok := mb[broker.SmID]; !ok {
			//TODO at some point we will be hitting platform rate limits... how should we handle that?
			updateOp(broker.ServiceBroker)
			affectedBrokers = append(affectedBrokers, broker)
		}
	}
	return affectedBrokers
}

func logFields(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"guid": broker.Guid,
		"name": broker.Name,
		"url":  broker.BrokerURL,
	}
}
