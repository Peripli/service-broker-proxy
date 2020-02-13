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
	"context"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// to match the desired broker scheduler coming from the Service Manager (desiredBrokers).
func (r *resyncJob) reconcileBrokers(ctx context.Context, existingBrokers, desiredBrokers []*platform.ServiceBroker) {
	brokerKeyMap := indexBrokers(existingBrokers, func(broker *platform.ServiceBroker) (string, bool) {
		return getBrokerKey(broker), true
	})
	proxyBrokerIDMap := indexBrokers(existingBrokers, func(broker *platform.ServiceBroker) (string, bool) {
		brokerID := brokerIDFromURL(broker.BrokerURL)
		if strings.HasPrefix(broker.BrokerURL, r.smPath) {
			return brokerID, true
		}

		if broker.BrokerURL == fmt.Sprintf(r.proxyPathPattern, brokerID) {
			return brokerID, true
		}

		return "", false
	})

	scheduler := NewScheduler(ctx, r.options.MaxParallelRequests)
	for _, desiredBroker := range desiredBrokers {
		desiredBroker := desiredBroker
		existingBroker, alreadyTakenOver := proxyBrokerIDMap[desiredBroker.GUID]
		delete(proxyBrokerIDMap, desiredBroker.GUID)

		if alreadyTakenOver {
			r.resyncTakenOverBroker(ctx, scheduler, desiredBroker, existingBroker)
		} else {
			r.resyncNotTakenOverBroker(ctx, scheduler, desiredBroker, brokerKeyMap)
		}
	}

	for _, existingBroker := range proxyBrokerIDMap {
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.deleteBrokerRegistration(ctx, existingBroker)
		}); err != nil {
			log.C(ctx).WithError(err).Error("resyncJob - could not delete broker registration from platform")
		}
	}
	if err := scheduler.Await(); err != nil {
		log.C(ctx).WithError(err).Error("resyncJob - could not reconcile brokers in platform")
	}
}

func (r *resyncJob) resyncNotTakenOverBroker(ctx context.Context, scheduler *TaskScheduler, desiredBroker *platform.ServiceBroker, brokerKeyMap map[string]*platform.ServiceBroker) {
	platformBroker, shouldBeTakenOver := brokerKeyMap[getBrokerKey(desiredBroker)]

	if shouldBeTakenOver {
		if r.options.TakeoverEnabled {
			if err := scheduler.Schedule(func(ctx context.Context) error {
				return r.updateBrokerRegistration(ctx, platformBroker.GUID, desiredBroker)
			}); err != nil {
				log.C(ctx).WithError(err).Error("resyncJob - could not update broker registration in platform")
			}
		}
	} else {
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.createBrokerRegistration(ctx, desiredBroker)
		}); err != nil {
			log.C(ctx).WithError(err).Error("resyncJob - could not create broker registration in platform")
		}
	}
}

func (r *resyncJob) resyncTakenOverBroker(ctx context.Context, scheduler *TaskScheduler, desiredBroker *platform.ServiceBroker, existingBroker *platform.ServiceBroker) {
	if existingBroker.Name != r.brokerProxyName(desiredBroker) || !strings.HasPrefix(existingBroker.BrokerURL, r.smPath) { // broker name has been changed in the platform or broker proxy URL should be updated
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.updateBrokerRegistration(ctx, existingBroker.GUID, desiredBroker)
		}); err != nil {
			log.C(ctx).WithError(err).Error("resyncJob - could not update broker registration in platform")
		}
	} else {
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.fetchBrokerCatalog(ctx, existingBroker)
		}); err != nil {
			log.C(ctx).WithError(err).Error("resyncJob - could not refetch broker catalog in platform")
		}
	}
}

func (r *resyncJob) getBrokersFromSM(ctx context.Context) ([]*platform.ServiceBroker, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting brokers from Service Manager...")

	proxyBrokers, err := r.smClient.GetBrokers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]*platform.ServiceBroker, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		if slice.StringsAnyEquals(r.options.BrokerBlacklist, broker.Name) {
			continue
		}

		brokerReg := &platform.ServiceBroker{
			GUID:      broker.ID,
			Name:      broker.Name,
			BrokerURL: broker.BrokerURL,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logger.Infof("resyncJob SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r *resyncJob) fetchBrokerCatalog(ctx context.Context, broker *platform.ServiceBroker) error {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(ctx)
		logger.WithFields(logBroker(broker)).Infof("resyncJob refetching catalog for broker...")
		if err := f.Fetch(ctx, broker); err != nil {
			logger.WithFields(logBroker(broker)).WithError(err).Error("Error during fetching catalog...")
			return err
		}
		logger.WithFields(logBroker(broker)).Info("resyncJob SUCCESSFULLY refetched catalog for broker")
	}
	return nil
}

func (r *resyncJob) createBrokerRegistration(ctx context.Context, broker *platform.ServiceBroker) error {
	logger := log.C(ctx)
	logger.WithFields(logBroker(broker)).Info("resyncJob creating proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      r.brokerProxyName(broker),
		BrokerURL: r.smPath + "/" + broker.GUID,
	}
	b, err := r.platformClient.Broker().CreateBroker(ctx, createRequest)
	if err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker creation")
		return err
	}
	logger.WithFields(logBroker(b)).Infof("resyncJob SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	return nil
}

func (r *resyncJob) updateBrokerRegistration(ctx context.Context, brokerGUID string, broker *platform.ServiceBroker) error {
	logger := log.C(ctx)

	logger.WithFields(logBroker(broker)).Info("resyncJob updating broker registration in platform...")

	updateRequest := &platform.UpdateServiceBrokerRequest{
		GUID:      brokerGUID,
		Name:      r.brokerProxyName(broker),
		BrokerURL: r.smPath + "/" + broker.GUID,
	}
	b, err := r.platformClient.Broker().UpdateBroker(ctx, updateRequest)
	if err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker update")
		return err
	}
	logger.WithFields(logBroker(b)).Infof("resyncJob SUCCESSFULLY updated broker registration at platform under name [%s] accessible at [%s]", updateRequest.Name, updateRequest.BrokerURL)
	return nil
}

func (r *resyncJob) deleteBrokerRegistration(ctx context.Context, broker *platform.ServiceBroker) error {
	logger := log.C(ctx)
	logger.WithFields(logBroker(broker)).Info("resyncJob deleting broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.Broker().DeleteBroker(ctx, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
		return err
	}
	logger.WithFields(logBroker(broker)).Infof("resyncJob SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	return nil
}

func (r *resyncJob) brokerProxyName(broker *platform.ServiceBroker) string {
	return fmt.Sprintf("%s%s-%s", r.options.BrokerPrefix, broker.Name, broker.GUID)
}

func logBroker(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.GUID,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}

func brokerIDFromURL(brokerURL string) string {
	return brokerURL[strings.LastIndex(brokerURL, "/")+1:]
}

func getBrokerKey(broker *platform.ServiceBroker) string {
	return fmt.Sprintf("name:%s|url:%s", broker.Name, strings.TrimRight(broker.BrokerURL, "/"))
}

func indexBrokers(brokers []*platform.ServiceBroker, indexingFunc func(broker *platform.ServiceBroker) (string, bool)) map[string]*platform.ServiceBroker {
	brokerMap := map[string]*platform.ServiceBroker{}
	for _, broker := range brokers {
		broker := broker
		if key, ok := indexingFunc(broker); ok {
			brokerMap[key] = broker
		}
	}
	return brokerMap
}
