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

package reconcile

import (
	"github.com/Peripli/service-manager/pkg/log"

	"strings"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type brokerKey struct {
	name string
	url  string
}

// processBrokers handles the reconciliation of the service brokers.
// it gets the brokers from SM and the platform and runs the reconciliation
func (r *Reconciler) processBrokers() {
	logger := log.C(r.runContext)
	if r.platformClient.Broker() == nil {
		logger.Debug("Platform client cannot handle brokers. Broker reconciliation will be skipped.")
		return
	}

	// get all the registered brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform()
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining already registered brokers")
		return
	}

	brokersFromSM, err := r.getBrokersFromSM()
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining brokers from Service Manager")
		return
	}
	r.stats[smBrokersStats] = brokersFromSM

	// control logic - make sure current state matches desired state
	r.reconcileBrokers(brokersFromPlatform, brokersFromSM)
}

// reconcileBrokers attempts to reconcile the current brokers state in the platform (existingBrokers)
// to match the desired broker state coming from the Service Manager (payloadBrokers).
func (r *Reconciler) reconcileBrokers(existingBrokers []platform.ServiceBroker, payloadBrokers []platform.ServiceBroker) {
	brokerKeyMap := indexBrokersByKey(existingBrokers)
	proxyBrokerIDMap := indexProxyBrokersByID(existingBrokers, r.proxyPath)

	for _, payloadBroker := range payloadBrokers {
		existingBroker := proxyBrokerIDMap[payloadBroker.GUID]
		delete(proxyBrokerIDMap, payloadBroker.GUID)
		platformBroker, knownToPlatform := brokerKeyMap[getBrokerKey(&payloadBroker)]
		knownToSM := existingBroker != nil
		// Broker is registered in platform, this is its first registration in SM but not already known to this broker proxy
		if knownToPlatform && !knownToSM {
			r.updateBrokerRegistration(platformBroker.GUID, &payloadBroker)
		} else if !knownToSM {
			r.createBrokerRegistration(&payloadBroker)
		} else {
			r.fetchBrokerCatalog(existingBroker)
		}
	}

	for _, existingBroker := range proxyBrokerIDMap {
		r.deleteBrokerRegistration(existingBroker)
	}
}

func (r *Reconciler) getBrokersFromPlatform() ([]platform.ServiceBroker, error) {
	logger := log.C(r.runContext)
	logger.Info("Reconciler getting brokers from platform...")
	registeredBrokers, err := r.platformClient.Broker().GetBrokers(r.runContext)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}
	logger.Debugf("Reconciler SUCCESSFULLY retrieved %d brokers from platform", len(registeredBrokers))

	return registeredBrokers, nil
}

func (r *Reconciler) getBrokersFromSM() ([]platform.ServiceBroker, error) {
	logger := log.C(r.runContext)
	logger.Info("Reconciler getting brokers from Service Manager...")

	proxyBrokers, err := r.smClient.GetBrokers(r.runContext)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]platform.ServiceBroker, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := platform.ServiceBroker{
			GUID:             broker.ID,
			Name:             broker.Name,
			BrokerURL:        broker.BrokerURL,
			ServiceOfferings: broker.ServiceOfferings,
			Metadata:         broker.Metadata,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logger.Infof("Reconciler SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r *Reconciler) fetchBrokerCatalog(broker *platform.ServiceBroker) {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(r.runContext)
		logger.WithFields(logBroker(broker)).Infof("Reconciler refetching catalog for broker...")
		if err := f.Fetch(r.runContext, broker); err != nil {
			logger.WithFields(logBroker(broker)).WithError(err).Error("Error during fetching catalog...")
		} else {
			logger.WithFields(logBroker(broker)).Info("Reconciler SUCCESSFULLY refetched catalog for broker")
		}
	}
}

func (r *Reconciler) createBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.runContext)
	logger.WithFields(logBroker(broker)).Info("Reconciler creating proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      r.options.BrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	if b, err := r.platformClient.Broker().CreateBroker(r.runContext, createRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker creation")
	} else {
		logger.WithFields(logBroker(b)).Infof("Reconciler SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	}
}

func (r *Reconciler) updateBrokerRegistration(brokerGUID string, broker *platform.ServiceBroker) {
	logger := log.C(r.runContext)
	logger.WithFields(logBroker(broker)).Info("Reconciler updating broker registration in platform...")

	updateRequest := &platform.UpdateServiceBrokerRequest{
		GUID:      brokerGUID,
		Name:      r.options.BrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	if b, err := r.platformClient.Broker().UpdateBroker(r.runContext, updateRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker update")
	} else {
		logger.WithFields(logBroker(b)).Infof("Reconciler SUCCESSFULLY updated broker registration at platform under name [%s] accessible at [%s]", updateRequest.Name, updateRequest.BrokerURL)
	}
}

func (r *Reconciler) deleteBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.runContext)
	logger.WithFields(logBroker(broker)).Info("Reconciler deleting broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.Broker().DeleteBroker(r.runContext, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
	} else {
		logger.WithFields(logBroker(broker)).Infof("Reconciler SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	}
}

func logBroker(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.GUID,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}

func getBrokerKey(broker *platform.ServiceBroker) brokerKey {
	return brokerKey{
		name: broker.Name,
		url:  broker.BrokerURL,
	}
}

func indexBrokersByKey(brokerList []platform.ServiceBroker) map[brokerKey]*platform.ServiceBroker {
	brokerMap := map[brokerKey]*platform.ServiceBroker{}
	for _, broker := range brokerList {
		broker := broker
		brokerMap[getBrokerKey(&broker)] = &broker
	}
	return brokerMap
}

func indexProxyBrokersByID(brokerList []platform.ServiceBroker, proxyPath string) map[string]*platform.ServiceBroker {
	brokerMap := map[string]*platform.ServiceBroker{}
	for _, broker := range brokerList {
		broker := broker
		if strings.HasPrefix(broker.BrokerURL, proxyPath) {
			brokerID := broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:]
			brokerMap[brokerID] = &broker
		}
	}
	return brokerMap
}
