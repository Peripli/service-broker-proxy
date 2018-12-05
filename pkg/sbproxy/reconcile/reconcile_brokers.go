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
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/log"

	"strings"

	"encoding/json"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ProxyBrokerPrefix prefixes names of brokers registered at the platform
const ProxyBrokerPrefix = "sm-proxy-"

// reconcileBrokers attempts to reconcile the current brokers state in the platform (existingBrokers)
// to match the desired broker state coming from the Service Manager (payloadBrokers).
func (r ReconcilationTask) reconcileBrokers(existingBrokers []platform.ServiceBroker, payloadBrokers []platform.ServiceBroker) bool {
	changes := false
	existingMap := convertBrokersRegListToMap(existingBrokers)
	for _, payloadBroker := range payloadBrokers {
		existingBroker := existingMap[payloadBroker.GUID]
		delete(existingMap, payloadBroker.GUID)

		// We don't care whether the following methods return errors
		// the errors are logged by the method itself. We just proceed
		if existingBroker == nil {
			if err := r.createBrokerRegistration(&payloadBroker); err == nil {
				changes = true
			}
		} else {
			r.fetchBrokerCatalog(existingBroker)
		}
	}

	for _, existingBroker := range existingMap {
		r.deleteBrokerRegistration(existingBroker)
		changes = true
	}

	return changes
}

func (r ReconcilationTask) getBrokersFromPlatform() ([]platform.ServiceBroker, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask task getting proxy brokers from platform...")
	registeredBrokers, err := r.platformClient.GetBrokers(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}

	brokersFromPlatform := make([]platform.ServiceBroker, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		if !r.isProxyBroker(broker) {
			continue
		}

		logger.WithFields(logBroker(&broker)).Debug("ReconcilationTask task FOUND registered proxy broker... ")
		brokersFromPlatform = append(brokersFromPlatform, broker)
	}
	logger.Debugf("ReconcilationTask task SUCCESSFULLY retrieved %d proxy brokers from platform", len(brokersFromPlatform))
	return brokersFromPlatform, nil
}

func (r ReconcilationTask) getBrokersFromSM() ([]platform.ServiceBroker, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask task getting brokers from Service Manager")

	proxyBrokers, err := r.smClient.GetBrokers(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]platform.ServiceBroker, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := platform.ServiceBroker{
			GUID:             broker.ID,
			BrokerURL:        broker.BrokerURL,
			ServiceOfferings: broker.ServiceOfferings,
			Metadata:         broker.Metadata,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logger.Debugf("ReconcilationTask task SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r ReconcilationTask) fetchBrokerCatalog(broker *platform.ServiceBroker) (err error) {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(r.ctx)
		logger.WithFields(logBroker(broker)).Debugf("ReconcilationTask task refetching catalog for broker")
		if err = f.Fetch(r.ctx, broker); err != nil {
			logger.WithFields(logBroker(broker)).WithError(err).Error("Error during fetching catalog...")
		} else {
			logger.WithFields(logBroker(broker)).Debug("ReconcilationTask task SUCCESSFULLY refetched catalog for broker")
		}
	}
	return
}

func (r ReconcilationTask) createBrokerRegistration(broker *platform.ServiceBroker) (err error) {
	logger := log.C(r.ctx)
	logger.WithFields(logBroker(broker)).Info("ReconcilationTask task attempting to create proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	var b *platform.ServiceBroker
	if b, err = r.platformClient.CreateBroker(r.ctx, createRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker creation")
	} else {
		logger.WithFields(logBroker(b)).Infof("ReconcilationTask task SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	}
	return
}

func (r ReconcilationTask) deleteBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.ctx)
	logger.WithFields(logBroker(broker)).Info("ReconcilationTask task attempting to delete broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.DeleteBroker(r.ctx, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
	} else {
		logger.WithFields(logBroker(broker)).Infof("ReconcilationTask task SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	}
}

func (r ReconcilationTask) isProxyBroker(broker platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyPath)
}

func logBroker(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.GUID,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}

func logService(service types.ServiceOffering) logrus.Fields {
	return logrus.Fields{
		"service_guid": service.CatalogID,
		"service_name": service.Name,
	}
}

func emptyContext() json.RawMessage {
	return json.RawMessage(`{}`)
}

func convertBrokersRegListToMap(brokerList []platform.ServiceBroker) map[string]*platform.ServiceBroker {
	brokerRegMap := make(map[string]*platform.ServiceBroker, len(brokerList))

	for i, broker := range brokerList {
		smID := broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:]
		brokerRegMap[smID] = &brokerList[i]
	}
	return brokerRegMap
}
