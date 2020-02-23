// Code generated by counterfeiter. DO NOT EDIT.
package smfakes

import (
	"context"
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
)

type FakeClient struct {
	DeleteCredentialsStub        func(context.Context, *types.BrokerPlatformCredential) error
	deleteCredentialsMutex       sync.RWMutex
	deleteCredentialsArgsForCall []struct {
		arg1 context.Context
		arg2 *types.BrokerPlatformCredential
	}
	deleteCredentialsReturns struct {
		result1 error
	}
	deleteCredentialsReturnsOnCall map[int]struct {
		result1 error
	}
	GetBrokersStub        func(context.Context) ([]*types.ServiceBroker, error)
	getBrokersMutex       sync.RWMutex
	getBrokersArgsForCall []struct {
		arg1 context.Context
	}
	getBrokersReturns struct {
		result1 []*types.ServiceBroker
		result2 error
	}
	getBrokersReturnsOnCall map[int]struct {
		result1 []*types.ServiceBroker
		result2 error
	}
	GetPlansStub        func(context.Context) ([]*types.ServicePlan, error)
	getPlansMutex       sync.RWMutex
	getPlansArgsForCall []struct {
		arg1 context.Context
	}
	getPlansReturns struct {
		result1 []*types.ServicePlan
		result2 error
	}
	getPlansReturnsOnCall map[int]struct {
		result1 []*types.ServicePlan
		result2 error
	}
	GetPlansByServiceOfferingsStub        func(context.Context, []*types.ServiceOffering) ([]*types.ServicePlan, error)
	getPlansByServiceOfferingsMutex       sync.RWMutex
	getPlansByServiceOfferingsArgsForCall []struct {
		arg1 context.Context
		arg2 []*types.ServiceOffering
	}
	getPlansByServiceOfferingsReturns struct {
		result1 []*types.ServicePlan
		result2 error
	}
	getPlansByServiceOfferingsReturnsOnCall map[int]struct {
		result1 []*types.ServicePlan
		result2 error
	}
	GetServiceOfferingsByBrokerIDsStub        func(context.Context, []string) ([]*types.ServiceOffering, error)
	getServiceOfferingsByBrokerIDsMutex       sync.RWMutex
	getServiceOfferingsByBrokerIDsArgsForCall []struct {
		arg1 context.Context
		arg2 []string
	}
	getServiceOfferingsByBrokerIDsReturns struct {
		result1 []*types.ServiceOffering
		result2 error
	}
	getServiceOfferingsByBrokerIDsReturnsOnCall map[int]struct {
		result1 []*types.ServiceOffering
		result2 error
	}
	GetVisibilitiesStub        func(context.Context) ([]*types.Visibility, error)
	getVisibilitiesMutex       sync.RWMutex
	getVisibilitiesArgsForCall []struct {
		arg1 context.Context
	}
	getVisibilitiesReturns struct {
		result1 []*types.Visibility
		result2 error
	}
	getVisibilitiesReturnsOnCall map[int]struct {
		result1 []*types.Visibility
		result2 error
	}
	RegisterCredentialsStub        func(context.Context, *types.BrokerPlatformCredential) error
	registerCredentialsMutex       sync.RWMutex
	registerCredentialsArgsForCall []struct {
		arg1 context.Context
		arg2 *types.BrokerPlatformCredential
	}
	registerCredentialsReturns struct {
		result1 error
	}
	registerCredentialsReturnsOnCall map[int]struct {
		result1 error
	}
	UpdateCredentialsStub        func(context.Context, *types.BrokerPlatformCredential) error
	updateCredentialsMutex       sync.RWMutex
	updateCredentialsArgsForCall []struct {
		arg1 context.Context
		arg2 *types.BrokerPlatformCredential
	}
	updateCredentialsReturns struct {
		result1 error
	}
	updateCredentialsReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeClient) DeleteCredentials(arg1 context.Context, arg2 *types.BrokerPlatformCredential) error {
	fake.deleteCredentialsMutex.Lock()
	ret, specificReturn := fake.deleteCredentialsReturnsOnCall[len(fake.deleteCredentialsArgsForCall)]
	fake.deleteCredentialsArgsForCall = append(fake.deleteCredentialsArgsForCall, struct {
		arg1 context.Context
		arg2 *types.BrokerPlatformCredential
	}{arg1, arg2})
	fake.recordInvocation("DeleteCredentials", []interface{}{arg1, arg2})
	fake.deleteCredentialsMutex.Unlock()
	if fake.DeleteCredentialsStub != nil {
		return fake.DeleteCredentialsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.deleteCredentialsReturns
	return fakeReturns.result1
}

func (fake *FakeClient) DeleteCredentialsCallCount() int {
	fake.deleteCredentialsMutex.RLock()
	defer fake.deleteCredentialsMutex.RUnlock()
	return len(fake.deleteCredentialsArgsForCall)
}

func (fake *FakeClient) DeleteCredentialsCalls(stub func(context.Context, *types.BrokerPlatformCredential) error) {
	fake.deleteCredentialsMutex.Lock()
	defer fake.deleteCredentialsMutex.Unlock()
	fake.DeleteCredentialsStub = stub
}

func (fake *FakeClient) DeleteCredentialsArgsForCall(i int) (context.Context, *types.BrokerPlatformCredential) {
	fake.deleteCredentialsMutex.RLock()
	defer fake.deleteCredentialsMutex.RUnlock()
	argsForCall := fake.deleteCredentialsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) DeleteCredentialsReturns(result1 error) {
	fake.deleteCredentialsMutex.Lock()
	defer fake.deleteCredentialsMutex.Unlock()
	fake.DeleteCredentialsStub = nil
	fake.deleteCredentialsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) DeleteCredentialsReturnsOnCall(i int, result1 error) {
	fake.deleteCredentialsMutex.Lock()
	defer fake.deleteCredentialsMutex.Unlock()
	fake.DeleteCredentialsStub = nil
	if fake.deleteCredentialsReturnsOnCall == nil {
		fake.deleteCredentialsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteCredentialsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) GetBrokers(arg1 context.Context) ([]*types.ServiceBroker, error) {
	fake.getBrokersMutex.Lock()
	ret, specificReturn := fake.getBrokersReturnsOnCall[len(fake.getBrokersArgsForCall)]
	fake.getBrokersArgsForCall = append(fake.getBrokersArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	fake.recordInvocation("GetBrokers", []interface{}{arg1})
	fake.getBrokersMutex.Unlock()
	if fake.GetBrokersStub != nil {
		return fake.GetBrokersStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getBrokersReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetBrokersCallCount() int {
	fake.getBrokersMutex.RLock()
	defer fake.getBrokersMutex.RUnlock()
	return len(fake.getBrokersArgsForCall)
}

func (fake *FakeClient) GetBrokersCalls(stub func(context.Context) ([]*types.ServiceBroker, error)) {
	fake.getBrokersMutex.Lock()
	defer fake.getBrokersMutex.Unlock()
	fake.GetBrokersStub = stub
}

func (fake *FakeClient) GetBrokersArgsForCall(i int) context.Context {
	fake.getBrokersMutex.RLock()
	defer fake.getBrokersMutex.RUnlock()
	argsForCall := fake.getBrokersArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) GetBrokersReturns(result1 []*types.ServiceBroker, result2 error) {
	fake.getBrokersMutex.Lock()
	defer fake.getBrokersMutex.Unlock()
	fake.GetBrokersStub = nil
	fake.getBrokersReturns = struct {
		result1 []*types.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetBrokersReturnsOnCall(i int, result1 []*types.ServiceBroker, result2 error) {
	fake.getBrokersMutex.Lock()
	defer fake.getBrokersMutex.Unlock()
	fake.GetBrokersStub = nil
	if fake.getBrokersReturnsOnCall == nil {
		fake.getBrokersReturnsOnCall = make(map[int]struct {
			result1 []*types.ServiceBroker
			result2 error
		})
	}
	fake.getBrokersReturnsOnCall[i] = struct {
		result1 []*types.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetPlans(arg1 context.Context) ([]*types.ServicePlan, error) {
	fake.getPlansMutex.Lock()
	ret, specificReturn := fake.getPlansReturnsOnCall[len(fake.getPlansArgsForCall)]
	fake.getPlansArgsForCall = append(fake.getPlansArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	fake.recordInvocation("GetPlans", []interface{}{arg1})
	fake.getPlansMutex.Unlock()
	if fake.GetPlansStub != nil {
		return fake.GetPlansStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getPlansReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetPlansCallCount() int {
	fake.getPlansMutex.RLock()
	defer fake.getPlansMutex.RUnlock()
	return len(fake.getPlansArgsForCall)
}

func (fake *FakeClient) GetPlansCalls(stub func(context.Context) ([]*types.ServicePlan, error)) {
	fake.getPlansMutex.Lock()
	defer fake.getPlansMutex.Unlock()
	fake.GetPlansStub = stub
}

func (fake *FakeClient) GetPlansArgsForCall(i int) context.Context {
	fake.getPlansMutex.RLock()
	defer fake.getPlansMutex.RUnlock()
	argsForCall := fake.getPlansArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) GetPlansReturns(result1 []*types.ServicePlan, result2 error) {
	fake.getPlansMutex.Lock()
	defer fake.getPlansMutex.Unlock()
	fake.GetPlansStub = nil
	fake.getPlansReturns = struct {
		result1 []*types.ServicePlan
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetPlansReturnsOnCall(i int, result1 []*types.ServicePlan, result2 error) {
	fake.getPlansMutex.Lock()
	defer fake.getPlansMutex.Unlock()
	fake.GetPlansStub = nil
	if fake.getPlansReturnsOnCall == nil {
		fake.getPlansReturnsOnCall = make(map[int]struct {
			result1 []*types.ServicePlan
			result2 error
		})
	}
	fake.getPlansReturnsOnCall[i] = struct {
		result1 []*types.ServicePlan
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetPlansByServiceOfferings(arg1 context.Context, arg2 []*types.ServiceOffering) ([]*types.ServicePlan, error) {
	var arg2Copy []*types.ServiceOffering
	if arg2 != nil {
		arg2Copy = make([]*types.ServiceOffering, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.getPlansByServiceOfferingsMutex.Lock()
	ret, specificReturn := fake.getPlansByServiceOfferingsReturnsOnCall[len(fake.getPlansByServiceOfferingsArgsForCall)]
	fake.getPlansByServiceOfferingsArgsForCall = append(fake.getPlansByServiceOfferingsArgsForCall, struct {
		arg1 context.Context
		arg2 []*types.ServiceOffering
	}{arg1, arg2Copy})
	fake.recordInvocation("GetPlansByServiceOfferings", []interface{}{arg1, arg2Copy})
	fake.getPlansByServiceOfferingsMutex.Unlock()
	if fake.GetPlansByServiceOfferingsStub != nil {
		return fake.GetPlansByServiceOfferingsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getPlansByServiceOfferingsReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetPlansByServiceOfferingsCallCount() int {
	fake.getPlansByServiceOfferingsMutex.RLock()
	defer fake.getPlansByServiceOfferingsMutex.RUnlock()
	return len(fake.getPlansByServiceOfferingsArgsForCall)
}

func (fake *FakeClient) GetPlansByServiceOfferingsCalls(stub func(context.Context, []*types.ServiceOffering) ([]*types.ServicePlan, error)) {
	fake.getPlansByServiceOfferingsMutex.Lock()
	defer fake.getPlansByServiceOfferingsMutex.Unlock()
	fake.GetPlansByServiceOfferingsStub = stub
}

func (fake *FakeClient) GetPlansByServiceOfferingsArgsForCall(i int) (context.Context, []*types.ServiceOffering) {
	fake.getPlansByServiceOfferingsMutex.RLock()
	defer fake.getPlansByServiceOfferingsMutex.RUnlock()
	argsForCall := fake.getPlansByServiceOfferingsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) GetPlansByServiceOfferingsReturns(result1 []*types.ServicePlan, result2 error) {
	fake.getPlansByServiceOfferingsMutex.Lock()
	defer fake.getPlansByServiceOfferingsMutex.Unlock()
	fake.GetPlansByServiceOfferingsStub = nil
	fake.getPlansByServiceOfferingsReturns = struct {
		result1 []*types.ServicePlan
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetPlansByServiceOfferingsReturnsOnCall(i int, result1 []*types.ServicePlan, result2 error) {
	fake.getPlansByServiceOfferingsMutex.Lock()
	defer fake.getPlansByServiceOfferingsMutex.Unlock()
	fake.GetPlansByServiceOfferingsStub = nil
	if fake.getPlansByServiceOfferingsReturnsOnCall == nil {
		fake.getPlansByServiceOfferingsReturnsOnCall = make(map[int]struct {
			result1 []*types.ServicePlan
			result2 error
		})
	}
	fake.getPlansByServiceOfferingsReturnsOnCall[i] = struct {
		result1 []*types.ServicePlan
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetServiceOfferingsByBrokerIDs(arg1 context.Context, arg2 []string) ([]*types.ServiceOffering, error) {
	var arg2Copy []string
	if arg2 != nil {
		arg2Copy = make([]string, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.getServiceOfferingsByBrokerIDsMutex.Lock()
	ret, specificReturn := fake.getServiceOfferingsByBrokerIDsReturnsOnCall[len(fake.getServiceOfferingsByBrokerIDsArgsForCall)]
	fake.getServiceOfferingsByBrokerIDsArgsForCall = append(fake.getServiceOfferingsByBrokerIDsArgsForCall, struct {
		arg1 context.Context
		arg2 []string
	}{arg1, arg2Copy})
	fake.recordInvocation("GetServiceOfferingsByBrokerIDs", []interface{}{arg1, arg2Copy})
	fake.getServiceOfferingsByBrokerIDsMutex.Unlock()
	if fake.GetServiceOfferingsByBrokerIDsStub != nil {
		return fake.GetServiceOfferingsByBrokerIDsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getServiceOfferingsByBrokerIDsReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetServiceOfferingsByBrokerIDsCallCount() int {
	fake.getServiceOfferingsByBrokerIDsMutex.RLock()
	defer fake.getServiceOfferingsByBrokerIDsMutex.RUnlock()
	return len(fake.getServiceOfferingsByBrokerIDsArgsForCall)
}

func (fake *FakeClient) GetServiceOfferingsByBrokerIDsCalls(stub func(context.Context, []string) ([]*types.ServiceOffering, error)) {
	fake.getServiceOfferingsByBrokerIDsMutex.Lock()
	defer fake.getServiceOfferingsByBrokerIDsMutex.Unlock()
	fake.GetServiceOfferingsByBrokerIDsStub = stub
}

func (fake *FakeClient) GetServiceOfferingsByBrokerIDsArgsForCall(i int) (context.Context, []string) {
	fake.getServiceOfferingsByBrokerIDsMutex.RLock()
	defer fake.getServiceOfferingsByBrokerIDsMutex.RUnlock()
	argsForCall := fake.getServiceOfferingsByBrokerIDsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) GetServiceOfferingsByBrokerIDsReturns(result1 []*types.ServiceOffering, result2 error) {
	fake.getServiceOfferingsByBrokerIDsMutex.Lock()
	defer fake.getServiceOfferingsByBrokerIDsMutex.Unlock()
	fake.GetServiceOfferingsByBrokerIDsStub = nil
	fake.getServiceOfferingsByBrokerIDsReturns = struct {
		result1 []*types.ServiceOffering
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetServiceOfferingsByBrokerIDsReturnsOnCall(i int, result1 []*types.ServiceOffering, result2 error) {
	fake.getServiceOfferingsByBrokerIDsMutex.Lock()
	defer fake.getServiceOfferingsByBrokerIDsMutex.Unlock()
	fake.GetServiceOfferingsByBrokerIDsStub = nil
	if fake.getServiceOfferingsByBrokerIDsReturnsOnCall == nil {
		fake.getServiceOfferingsByBrokerIDsReturnsOnCall = make(map[int]struct {
			result1 []*types.ServiceOffering
			result2 error
		})
	}
	fake.getServiceOfferingsByBrokerIDsReturnsOnCall[i] = struct {
		result1 []*types.ServiceOffering
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetVisibilities(arg1 context.Context) ([]*types.Visibility, error) {
	fake.getVisibilitiesMutex.Lock()
	ret, specificReturn := fake.getVisibilitiesReturnsOnCall[len(fake.getVisibilitiesArgsForCall)]
	fake.getVisibilitiesArgsForCall = append(fake.getVisibilitiesArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	fake.recordInvocation("GetVisibilities", []interface{}{arg1})
	fake.getVisibilitiesMutex.Unlock()
	if fake.GetVisibilitiesStub != nil {
		return fake.GetVisibilitiesStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getVisibilitiesReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) GetVisibilitiesCallCount() int {
	fake.getVisibilitiesMutex.RLock()
	defer fake.getVisibilitiesMutex.RUnlock()
	return len(fake.getVisibilitiesArgsForCall)
}

func (fake *FakeClient) GetVisibilitiesCalls(stub func(context.Context) ([]*types.Visibility, error)) {
	fake.getVisibilitiesMutex.Lock()
	defer fake.getVisibilitiesMutex.Unlock()
	fake.GetVisibilitiesStub = stub
}

func (fake *FakeClient) GetVisibilitiesArgsForCall(i int) context.Context {
	fake.getVisibilitiesMutex.RLock()
	defer fake.getVisibilitiesMutex.RUnlock()
	argsForCall := fake.getVisibilitiesArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeClient) GetVisibilitiesReturns(result1 []*types.Visibility, result2 error) {
	fake.getVisibilitiesMutex.Lock()
	defer fake.getVisibilitiesMutex.Unlock()
	fake.GetVisibilitiesStub = nil
	fake.getVisibilitiesReturns = struct {
		result1 []*types.Visibility
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetVisibilitiesReturnsOnCall(i int, result1 []*types.Visibility, result2 error) {
	fake.getVisibilitiesMutex.Lock()
	defer fake.getVisibilitiesMutex.Unlock()
	fake.GetVisibilitiesStub = nil
	if fake.getVisibilitiesReturnsOnCall == nil {
		fake.getVisibilitiesReturnsOnCall = make(map[int]struct {
			result1 []*types.Visibility
			result2 error
		})
	}
	fake.getVisibilitiesReturnsOnCall[i] = struct {
		result1 []*types.Visibility
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) RegisterCredentials(arg1 context.Context, arg2 *types.BrokerPlatformCredential) error {
	fake.registerCredentialsMutex.Lock()
	ret, specificReturn := fake.registerCredentialsReturnsOnCall[len(fake.registerCredentialsArgsForCall)]
	fake.registerCredentialsArgsForCall = append(fake.registerCredentialsArgsForCall, struct {
		arg1 context.Context
		arg2 *types.BrokerPlatformCredential
	}{arg1, arg2})
	fake.recordInvocation("RegisterCredentials", []interface{}{arg1, arg2})
	fake.registerCredentialsMutex.Unlock()
	if fake.RegisterCredentialsStub != nil {
		return fake.RegisterCredentialsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.registerCredentialsReturns
	return fakeReturns.result1
}

func (fake *FakeClient) RegisterCredentialsCallCount() int {
	fake.registerCredentialsMutex.RLock()
	defer fake.registerCredentialsMutex.RUnlock()
	return len(fake.registerCredentialsArgsForCall)
}

func (fake *FakeClient) RegisterCredentialsCalls(stub func(context.Context, *types.BrokerPlatformCredential) error) {
	fake.registerCredentialsMutex.Lock()
	defer fake.registerCredentialsMutex.Unlock()
	fake.RegisterCredentialsStub = stub
}

func (fake *FakeClient) RegisterCredentialsArgsForCall(i int) (context.Context, *types.BrokerPlatformCredential) {
	fake.registerCredentialsMutex.RLock()
	defer fake.registerCredentialsMutex.RUnlock()
	argsForCall := fake.registerCredentialsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) RegisterCredentialsReturns(result1 error) {
	fake.registerCredentialsMutex.Lock()
	defer fake.registerCredentialsMutex.Unlock()
	fake.RegisterCredentialsStub = nil
	fake.registerCredentialsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) RegisterCredentialsReturnsOnCall(i int, result1 error) {
	fake.registerCredentialsMutex.Lock()
	defer fake.registerCredentialsMutex.Unlock()
	fake.RegisterCredentialsStub = nil
	if fake.registerCredentialsReturnsOnCall == nil {
		fake.registerCredentialsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.registerCredentialsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) UpdateCredentials(arg1 context.Context, arg2 *types.BrokerPlatformCredential) error {
	fake.updateCredentialsMutex.Lock()
	ret, specificReturn := fake.updateCredentialsReturnsOnCall[len(fake.updateCredentialsArgsForCall)]
	fake.updateCredentialsArgsForCall = append(fake.updateCredentialsArgsForCall, struct {
		arg1 context.Context
		arg2 *types.BrokerPlatformCredential
	}{arg1, arg2})
	fake.recordInvocation("UpdateCredentials", []interface{}{arg1, arg2})
	fake.updateCredentialsMutex.Unlock()
	if fake.UpdateCredentialsStub != nil {
		return fake.UpdateCredentialsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.updateCredentialsReturns
	return fakeReturns.result1
}

func (fake *FakeClient) UpdateCredentialsCallCount() int {
	fake.updateCredentialsMutex.RLock()
	defer fake.updateCredentialsMutex.RUnlock()
	return len(fake.updateCredentialsArgsForCall)
}

func (fake *FakeClient) UpdateCredentialsCalls(stub func(context.Context, *types.BrokerPlatformCredential) error) {
	fake.updateCredentialsMutex.Lock()
	defer fake.updateCredentialsMutex.Unlock()
	fake.UpdateCredentialsStub = stub
}

func (fake *FakeClient) UpdateCredentialsArgsForCall(i int) (context.Context, *types.BrokerPlatformCredential) {
	fake.updateCredentialsMutex.RLock()
	defer fake.updateCredentialsMutex.RUnlock()
	argsForCall := fake.updateCredentialsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) UpdateCredentialsReturns(result1 error) {
	fake.updateCredentialsMutex.Lock()
	defer fake.updateCredentialsMutex.Unlock()
	fake.UpdateCredentialsStub = nil
	fake.updateCredentialsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) UpdateCredentialsReturnsOnCall(i int, result1 error) {
	fake.updateCredentialsMutex.Lock()
	defer fake.updateCredentialsMutex.Unlock()
	fake.UpdateCredentialsStub = nil
	if fake.updateCredentialsReturnsOnCall == nil {
		fake.updateCredentialsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateCredentialsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.deleteCredentialsMutex.RLock()
	defer fake.deleteCredentialsMutex.RUnlock()
	fake.getBrokersMutex.RLock()
	defer fake.getBrokersMutex.RUnlock()
	fake.getPlansMutex.RLock()
	defer fake.getPlansMutex.RUnlock()
	fake.getPlansByServiceOfferingsMutex.RLock()
	defer fake.getPlansByServiceOfferingsMutex.RUnlock()
	fake.getServiceOfferingsByBrokerIDsMutex.RLock()
	defer fake.getServiceOfferingsByBrokerIDsMutex.RUnlock()
	fake.getVisibilitiesMutex.RLock()
	defer fake.getVisibilitiesMutex.RUnlock()
	fake.registerCredentialsMutex.RLock()
	defer fake.registerCredentialsMutex.RUnlock()
	fake.updateCredentialsMutex.RLock()
	defer fake.updateCredentialsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeClient) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ sm.Client = new(FakeClient)
