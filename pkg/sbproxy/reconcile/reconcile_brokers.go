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
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/utils"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/util"
	"github.com/Peripli/service-manager/pkg/types"
	"strings"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// to match the desired broker state coming from the Service Manager (desiredBrokers).
func (r *resyncJob) reconcileBrokers(ctx context.Context, existingBrokers, desiredBrokers []*platform.ServiceBroker) {
	log.C(ctx).Infof("reconcile %v platform brokers and %v SM brokers", len(existingBrokers), len(desiredBrokers))

	pavelMap := map[string]*platform.ServiceBroker{}
	for _, broker := range existingBrokers {
		key := getBrokerKey(broker)
		pavelMap[key] = broker
	}
	log.C(ctx).Infof("pavel brokers map %v", pavelMap)

	brokerKeyMap := indexBrokers(existingBrokers, func(broker *platform.ServiceBroker) (string, bool) {
		return getBrokerKey(broker), true
	})
	log.C(ctx).Infof("brokers map %v", brokerKeyMap)

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

	log.C(ctx).Infof("taken over brokers map %v", proxyBrokerIDMap)

	scheduler := NewScheduler(ctx, r.options.MaxParallelRequests)
	for _, desiredBroker := range desiredBrokers {
		desiredBroker := desiredBroker
		existingBroker, alreadyTakenOver := proxyBrokerIDMap[desiredBroker.GUID]
		delete(proxyBrokerIDMap, desiredBroker.GUID)

		if alreadyTakenOver {
			log.C(ctx).Infof("%s already taken over", desiredBroker.GUID)
			r.resyncTakenOverBroker(ctx, scheduler, desiredBroker, existingBroker)
		} else {
			log.C(ctx).Infof("%s not taken over??", desiredBroker.GUID)
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
	brokerProxyName := utils.BrokerProxyName(r.platformClient, desiredBroker.Name, desiredBroker.GUID, r.options.BrokerPrefix)

	if shouldBeTakenOver {
		if r.options.TakeoverEnabled {
			if err := scheduler.Schedule(func(ctx context.Context) error {
				return r.updateBrokerRegistration(ctx, platformBroker.GUID, desiredBroker, brokerProxyName)
			}); err != nil {
				log.C(ctx).WithError(err).Error("resyncJob - could not update broker registration in platform")
			}
		}
	} else {
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.createBrokerRegistration(ctx, desiredBroker, brokerProxyName)
		}); err != nil {
			log.C(ctx).WithError(err).Error("resyncJob - could not create broker registration in platform")
		}
	}
}

func (r *resyncJob) resyncTakenOverBroker(ctx context.Context, scheduler *TaskScheduler, desiredBroker *platform.ServiceBroker, existingBroker *platform.ServiceBroker) {
	brokerProxyName := utils.BrokerProxyName(r.platformClient, desiredBroker.Name, desiredBroker.GUID, r.options.BrokerPrefix)
	// if broker name has been changed in the platform or broker proxy URL should be updated
	if existingBroker.Name != brokerProxyName || !strings.HasPrefix(existingBroker.BrokerURL, r.smPath) {
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.updateBrokerRegistration(ctx, existingBroker.GUID, desiredBroker, brokerProxyName)
		}); err != nil {
			log.C(ctx).WithError(err).Error("resyncJob - could not update broker registration in platform")
		}
	} else {
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.fetchBrokerCatalog(ctx, existingBroker.GUID, desiredBroker, brokerProxyName)
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
	logger.Infof("resyncJob successfully retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r *resyncJob) fetchBrokerCatalog(ctx context.Context, brokerGUIDInPlatform string, brokerInSM *platform.ServiceBroker, brokerProxyName string) error {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(ctx)
		logger.WithFields(logBroker(brokerInSM)).Infof("resyncJob refetching catalog for broker...")

		var username, password, passwordHash string
		var err error
		var credentialsResponse *types.BrokerPlatformCredential

		if r.options.BrokerCredentialsEnabled {
			username, password, passwordHash, err = util.GenerateBrokerPlatformCredentials()
			if err != nil {
				return fmt.Errorf("could not generate broker platform credentials for broker (%s): %s", brokerInSM.Name, err)
			}

			credentials := &types.BrokerPlatformCredential{
				Username:     username,
				PasswordHash: passwordHash,
				BrokerID:     brokerInSM.GUID,
			}

			credentialsResponse, err = r.smClient.PutCredentials(ctx, credentials, false)
			if err != nil {
				if err != sm.ErrConflictingBrokerPlatformCredentials {
					return fmt.Errorf("could not update broker platform credentials for broker (%s): %s", brokerInSM.Name, err)
				}
				username = ""
				password = ""
			}
		} else {
			username = r.defaultBrokerUsername
			password = r.defaultBrokerPassword
		}

		updateRequest := &platform.UpdateServiceBrokerRequest{
			ID:        brokerInSM.GUID,
			GUID:      brokerGUIDInPlatform,
			Name:      brokerProxyName,
			BrokerURL: r.smPath + "/" + brokerInSM.GUID,
			Username:  username,
			Password:  password,
		}

		if err := f.Fetch(ctx, updateRequest); err != nil {
			logger.WithFields(logBroker(brokerInSM)).WithError(err).Error("Error during fetching catalog...")
			return err
		}
		if r.options.BrokerCredentialsEnabled && len(username) > 0 && len(password) > 0 {
			r.activateBrokerCredentials(ctx, credentialsResponse)
		}
		logger.WithFields(logBroker(brokerInSM)).Info("resyncJob successfully re-fetched catalog for broker")
	}
	return nil
}

func (r *resyncJob) createBrokerRegistration(ctx context.Context, brokerInSM *platform.ServiceBroker, brokerProxyName string) error {
	logger := log.C(ctx)
	logger.WithFields(logBroker(brokerInSM)).Info("resyncJob creating proxy for broker in platform...")

	var username, password, passwordHash string
	var err error
	var credentialResponse *types.BrokerPlatformCredential

	if r.options.BrokerCredentialsEnabled {
		username, password, passwordHash, err = util.GenerateBrokerPlatformCredentials()
		if err != nil {
			return err
		}

		credentialResponse, err = r.smClient.PutCredentials(ctx, &types.BrokerPlatformCredential{
			Username:     username,
			PasswordHash: passwordHash,
			BrokerID:     brokerInSM.GUID,
		}, true)
		if err != nil {
			return err
		}
	} else {
		username = r.defaultBrokerUsername
		password = r.defaultBrokerPassword
	}

	createRequest := &platform.CreateServiceBrokerRequest{
		ID:        brokerInSM.GUID,
		Name:      brokerProxyName,
		BrokerURL: r.smPath + "/" + brokerInSM.GUID,
		Username:  username,
		Password:  password,
	}
	b, err := r.platformClient.Broker().CreateBroker(ctx, createRequest)
	if err != nil {
		logger.WithFields(logBroker(brokerInSM)).WithError(err).Error("Error during broker creation")
		return err
	}
	logger.WithFields(logBroker(b)).Infof("resyncJob successfully created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)

	if r.options.BrokerCredentialsEnabled {
		r.activateBrokerCredentials(ctx, credentialResponse)
	}

	return nil
}

func (r *resyncJob) updateBrokerRegistration(ctx context.Context, brokerGUIDInPlatform string, brokerInSM *platform.ServiceBroker, brokerProxyName string) error {
	logger := log.C(ctx)

	logger.WithFields(logBroker(brokerInSM)).Info("resyncJob updating broker registration in platform...")

	var username, password, passwordHash string
	var err error
	var credentialResponse *types.BrokerPlatformCredential

	if r.options.BrokerCredentialsEnabled {
		username, password, passwordHash, err = util.GenerateBrokerPlatformCredentials()
		if err != nil {
			return err
		}

		credentialResponse, err = r.smClient.PutCredentials(ctx, &types.BrokerPlatformCredential{
			Username:     username,
			PasswordHash: passwordHash,
			BrokerID:     brokerInSM.GUID,
		}, false)
		if err != nil {
			if err != sm.ErrConflictingBrokerPlatformCredentials {
				return fmt.Errorf("could not update broker platform credentials for broker (%s): %s", brokerInSM.Name, err)
			}
			username = ""
			password = ""
		}
	} else {
		username = r.defaultBrokerUsername
		password = r.defaultBrokerPassword
	}

	updateRequest := &platform.UpdateServiceBrokerRequest{
		ID:        brokerInSM.GUID,
		GUID:      brokerGUIDInPlatform,
		Name:      brokerProxyName,
		BrokerURL: r.smPath + "/" + brokerInSM.GUID,
		Username:  username,
		Password:  password,
	}
	b, err := r.platformClient.Broker().UpdateBroker(ctx, updateRequest)
	if err != nil {
		logger.WithFields(logBroker(brokerInSM)).WithError(err).Error("Error during broker update")
		return err
	}
	if r.options.BrokerCredentialsEnabled && len(username) > 0 && len(password) > 0 {
		r.activateBrokerCredentials(ctx, credentialResponse)
	}
	logger.WithFields(logBroker(b)).Infof("resyncJob successfully updated broker registration at platform under name [%s] accessible at [%s]", updateRequest.Name, updateRequest.BrokerURL)
	return nil
}

func (r *resyncJob) deleteBrokerRegistration(ctx context.Context, broker *platform.ServiceBroker) error {
	logger := log.C(ctx)
	logger.WithFields(logBroker(broker)).Info("resyncJob deleting broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		ID:   brokerIDFromURL(broker.BrokerURL),
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.Broker().DeleteBroker(ctx, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
		return err
	}

	logger.WithFields(logBroker(broker)).Infof("resyncJob successfully deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	return nil
}

func (r *resyncJob) activateBrokerCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) {
	log.C(ctx).Debugf("activating broker platform credentials with id %s", credentials.ID)
	err := r.smClient.ActivateCredentials(ctx, credentials.ID)
	if err != nil {
		log.C(ctx).WithError(err).Errorf("failed to activate credentials with id %s", credentials.ID)
	}
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
