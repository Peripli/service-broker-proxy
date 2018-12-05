// Code generated by counterfeiter. DO NOT EDIT.
package platformfakes

import (
	context "context"
	sync "sync"

	platform "github.com/Peripli/service-broker-proxy/pkg/platform"
)

type FakeClient struct {
	CreateBrokerStub        func(context.Context, *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error)
	createBrokerMutex       sync.RWMutex
	createBrokerArgsForCall []struct {
		arg1 context.Context
		arg2 *platform.CreateServiceBrokerRequest
	}
	createBrokerReturns struct {
		result1 *platform.ServiceBroker
		result2 error
	}
	createBrokerReturnsOnCall map[int]struct {
		result1 *platform.ServiceBroker
		result2 error
	}
	DeleteBrokerStub        func(context.Context, *platform.DeleteServiceBrokerRequest) error
	deleteBrokerMutex       sync.RWMutex
	deleteBrokerArgsForCall []struct {
		arg1 context.Context
		arg2 *platform.DeleteServiceBrokerRequest
	}
	deleteBrokerReturns struct {
		result1 error
	}
	deleteBrokerReturnsOnCall map[int]struct {
		result1 error
	}
	GetBrokersStub        func(context.Context) ([]platform.ServiceBroker, error)
	getBrokersMutex       sync.RWMutex
	getBrokersArgsForCall []struct {
		arg1 context.Context
	}
	getBrokersReturns struct {
		result1 []platform.ServiceBroker
		result2 error
	}
	getBrokersReturnsOnCall map[int]struct {
		result1 []platform.ServiceBroker
		result2 error
	}
	UpdateBrokerStub        func(context.Context, *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error)
	updateBrokerMutex       sync.RWMutex
	updateBrokerArgsForCall []struct {
		arg1 context.Context
		arg2 *platform.UpdateServiceBrokerRequest
	}
	updateBrokerReturns struct {
		result1 *platform.ServiceBroker
		result2 error
	}
	updateBrokerReturnsOnCall map[int]struct {
		result1 *platform.ServiceBroker
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeClient) CreateBroker(arg1 context.Context, arg2 *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	fake.createBrokerMutex.Lock()
	ret, specificReturn := fake.createBrokerReturnsOnCall[len(fake.createBrokerArgsForCall)]
	fake.createBrokerArgsForCall = append(fake.createBrokerArgsForCall, struct {
		arg1 context.Context
		arg2 *platform.CreateServiceBrokerRequest
	}{arg1, arg2})
	fake.recordInvocation("CreateBroker", []interface{}{arg1, arg2})
	fake.createBrokerMutex.Unlock()
	if fake.CreateBrokerStub != nil {
		return fake.CreateBrokerStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.createBrokerReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) CreateBrokerCallCount() int {
	fake.createBrokerMutex.RLock()
	defer fake.createBrokerMutex.RUnlock()
	return len(fake.createBrokerArgsForCall)
}

func (fake *FakeClient) CreateBrokerCalls(stub func(context.Context, *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error)) {
	fake.createBrokerMutex.Lock()
	defer fake.createBrokerMutex.Unlock()
	fake.CreateBrokerStub = stub
}

func (fake *FakeClient) CreateBrokerArgsForCall(i int) (context.Context, *platform.CreateServiceBrokerRequest) {
	fake.createBrokerMutex.RLock()
	defer fake.createBrokerMutex.RUnlock()
	argsForCall := fake.createBrokerArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) CreateBrokerReturns(result1 *platform.ServiceBroker, result2 error) {
	fake.createBrokerMutex.Lock()
	defer fake.createBrokerMutex.Unlock()
	fake.CreateBrokerStub = nil
	fake.createBrokerReturns = struct {
		result1 *platform.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) CreateBrokerReturnsOnCall(i int, result1 *platform.ServiceBroker, result2 error) {
	fake.createBrokerMutex.Lock()
	defer fake.createBrokerMutex.Unlock()
	fake.CreateBrokerStub = nil
	if fake.createBrokerReturnsOnCall == nil {
		fake.createBrokerReturnsOnCall = make(map[int]struct {
			result1 *platform.ServiceBroker
			result2 error
		})
	}
	fake.createBrokerReturnsOnCall[i] = struct {
		result1 *platform.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) DeleteBroker(arg1 context.Context, arg2 *platform.DeleteServiceBrokerRequest) error {
	fake.deleteBrokerMutex.Lock()
	ret, specificReturn := fake.deleteBrokerReturnsOnCall[len(fake.deleteBrokerArgsForCall)]
	fake.deleteBrokerArgsForCall = append(fake.deleteBrokerArgsForCall, struct {
		arg1 context.Context
		arg2 *platform.DeleteServiceBrokerRequest
	}{arg1, arg2})
	fake.recordInvocation("DeleteBroker", []interface{}{arg1, arg2})
	fake.deleteBrokerMutex.Unlock()
	if fake.DeleteBrokerStub != nil {
		return fake.DeleteBrokerStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.deleteBrokerReturns
	return fakeReturns.result1
}

func (fake *FakeClient) DeleteBrokerCallCount() int {
	fake.deleteBrokerMutex.RLock()
	defer fake.deleteBrokerMutex.RUnlock()
	return len(fake.deleteBrokerArgsForCall)
}

func (fake *FakeClient) DeleteBrokerCalls(stub func(context.Context, *platform.DeleteServiceBrokerRequest) error) {
	fake.deleteBrokerMutex.Lock()
	defer fake.deleteBrokerMutex.Unlock()
	fake.DeleteBrokerStub = stub
}

func (fake *FakeClient) DeleteBrokerArgsForCall(i int) (context.Context, *platform.DeleteServiceBrokerRequest) {
	fake.deleteBrokerMutex.RLock()
	defer fake.deleteBrokerMutex.RUnlock()
	argsForCall := fake.deleteBrokerArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) DeleteBrokerReturns(result1 error) {
	fake.deleteBrokerMutex.Lock()
	defer fake.deleteBrokerMutex.Unlock()
	fake.DeleteBrokerStub = nil
	fake.deleteBrokerReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) DeleteBrokerReturnsOnCall(i int, result1 error) {
	fake.deleteBrokerMutex.Lock()
	defer fake.deleteBrokerMutex.Unlock()
	fake.DeleteBrokerStub = nil
	if fake.deleteBrokerReturnsOnCall == nil {
		fake.deleteBrokerReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.deleteBrokerReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeClient) GetBrokers(arg1 context.Context) ([]platform.ServiceBroker, error) {
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

func (fake *FakeClient) GetBrokersCalls(stub func(context.Context) ([]platform.ServiceBroker, error)) {
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

func (fake *FakeClient) GetBrokersReturns(result1 []platform.ServiceBroker, result2 error) {
	fake.getBrokersMutex.Lock()
	defer fake.getBrokersMutex.Unlock()
	fake.GetBrokersStub = nil
	fake.getBrokersReturns = struct {
		result1 []platform.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetBrokersReturnsOnCall(i int, result1 []platform.ServiceBroker, result2 error) {
	fake.getBrokersMutex.Lock()
	defer fake.getBrokersMutex.Unlock()
	fake.GetBrokersStub = nil
	if fake.getBrokersReturnsOnCall == nil {
		fake.getBrokersReturnsOnCall = make(map[int]struct {
			result1 []platform.ServiceBroker
			result2 error
		})
	}
	fake.getBrokersReturnsOnCall[i] = struct {
		result1 []platform.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) UpdateBroker(arg1 context.Context, arg2 *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	fake.updateBrokerMutex.Lock()
	ret, specificReturn := fake.updateBrokerReturnsOnCall[len(fake.updateBrokerArgsForCall)]
	fake.updateBrokerArgsForCall = append(fake.updateBrokerArgsForCall, struct {
		arg1 context.Context
		arg2 *platform.UpdateServiceBrokerRequest
	}{arg1, arg2})
	fake.recordInvocation("UpdateBroker", []interface{}{arg1, arg2})
	fake.updateBrokerMutex.Unlock()
	if fake.UpdateBrokerStub != nil {
		return fake.UpdateBrokerStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.updateBrokerReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeClient) UpdateBrokerCallCount() int {
	fake.updateBrokerMutex.RLock()
	defer fake.updateBrokerMutex.RUnlock()
	return len(fake.updateBrokerArgsForCall)
}

func (fake *FakeClient) UpdateBrokerCalls(stub func(context.Context, *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error)) {
	fake.updateBrokerMutex.Lock()
	defer fake.updateBrokerMutex.Unlock()
	fake.UpdateBrokerStub = stub
}

func (fake *FakeClient) UpdateBrokerArgsForCall(i int) (context.Context, *platform.UpdateServiceBrokerRequest) {
	fake.updateBrokerMutex.RLock()
	defer fake.updateBrokerMutex.RUnlock()
	argsForCall := fake.updateBrokerArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeClient) UpdateBrokerReturns(result1 *platform.ServiceBroker, result2 error) {
	fake.updateBrokerMutex.Lock()
	defer fake.updateBrokerMutex.Unlock()
	fake.UpdateBrokerStub = nil
	fake.updateBrokerReturns = struct {
		result1 *platform.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) UpdateBrokerReturnsOnCall(i int, result1 *platform.ServiceBroker, result2 error) {
	fake.updateBrokerMutex.Lock()
	defer fake.updateBrokerMutex.Unlock()
	fake.UpdateBrokerStub = nil
	if fake.updateBrokerReturnsOnCall == nil {
		fake.updateBrokerReturnsOnCall = make(map[int]struct {
			result1 *platform.ServiceBroker
			result2 error
		})
	}
	fake.updateBrokerReturnsOnCall[i] = struct {
		result1 *platform.ServiceBroker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createBrokerMutex.RLock()
	defer fake.createBrokerMutex.RUnlock()
	fake.deleteBrokerMutex.RLock()
	defer fake.deleteBrokerMutex.RUnlock()
	fake.getBrokersMutex.RLock()
	defer fake.getBrokersMutex.RUnlock()
	fake.updateBrokerMutex.RLock()
	defer fake.updateBrokerMutex.RUnlock()
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

var _ platform.Client = new(FakeClient)
