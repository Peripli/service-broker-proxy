package task

import (
	"reflect"

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
	newBrokers, err := r.smClient.GetBrokers()
	for _, broker := range newBrokers.ServiceBrokers {
		broker.Username = r.details.User
		broker.Password = r.details.Password
		broker.BrokerURL = r.details.Host + "/" + broker.Guid
	}
	if err != nil {
		logrus.Error("An error occurred while obtaining brokers that have to be registered at the platform: ", err)
		return
	}

	brokersToRegister, brokersToUpdate := difference(newBrokers.ServiceBrokers, registeredBrokers.ServiceBrokers)
	brokersToDelete, _ := difference(registeredBrokers.ServiceBrokers, newBrokers.ServiceBrokers)

	r.createBrokerRegistrations(brokersToRegister)
	r.updateBrokerRegistrations(brokersToUpdate)
	r.deleteBrokerRegistrations(brokersToDelete)
}

func (r SBProxyRegistration) deleteBrokerRegistrations(brokersToDelete []platform.ServiceBroker) {
	for _, broker := range brokersToDelete {
		deleteRequest := &platform.DeleteServiceBrokerRequest{Guid: broker.Guid}
		if err := r.platformClient.DeleteBrokers(deleteRequest); err != nil {
			logrus.Error("Error during broker Update: ", err)
			//TODO on delete failure hook? what should we do if delete fails
		}
	}
}
func (r SBProxyRegistration) updateBrokerRegistrations(brokersToUpdate []platform.ServiceBroker) {
	for _, broker := range brokersToUpdate {
		updateRequest := &platform.UpdateServiceBrokerRequest{
			Guid:      broker.Guid,
			Name:      broker.Name,
			Username:  broker.Username,
			Password:  broker.Password,
			BrokerURL: broker.BrokerURL,
		}
		if _, err := r.platformClient.UpdateBrokers(updateRequest); err != nil {
			logrus.Error("Error during broker Update: ", err)
			//TODO on update failure hook? what should we do if update fails
		}
	}
}
func (r SBProxyRegistration) createBrokerRegistrations(brokersToRegister []platform.ServiceBroker) {
	for _, broker := range brokersToRegister {
		createRequest := &platform.CreateServiceBrokerRequest{Name: broker.Name,
			Username:  broker.Username,
			Password:  broker.Password,
			BrokerURL: broker.BrokerURL}
		if _, err := r.platformClient.CreateBrokers(createRequest); err != nil {
			logrus.Error("Error during broker registration: ", err)
			//TODO on registration failure hook? what should we do if reg fails
		}
	}
}

func difference(a, b []platform.ServiceBroker) ([]platform.ServiceBroker, []platform.ServiceBroker) {
	mb := make(map[string]platform.ServiceBroker)
	newBrokers := make([]platform.ServiceBroker, len(a))
	modifiedBrokers := make([]platform.ServiceBroker, len(a))

	for _, broker := range b {
		mb[broker.Guid] = broker
	}
	for _, broker := range a {
		if _, ok := mb[broker.Guid]; !ok {
			newBrokers = append(newBrokers, broker)
		}
		if val, ok := mb[broker.Guid]; ok && !reflect.DeepEqual(val, broker) {
			modifiedBrokers = append(modifiedBrokers, broker)
		}
	}
	return newBrokers, modifiedBrokers
}
