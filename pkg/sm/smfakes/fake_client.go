// Code generated by counterfeiter. DO NOT EDIT.
package smfakes

import (
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/sm"
)

type FakeClient struct {
	GetBrokersStub        func() ([]sm.Broker, error)
	getBrokersMutex       sync.RWMutex
	getBrokersArgsForCall []struct{}
	getBrokersReturns     struct {
		result1 []sm.Broker
		result2 error
	}
	getBrokersReturnsOnCall map[int]struct {
		result1 []sm.Broker
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeClient) GetBrokers() ([]sm.Broker, error) {
	fake.getBrokersMutex.Lock()
	ret, specificReturn := fake.getBrokersReturnsOnCall[len(fake.getBrokersArgsForCall)]
	fake.getBrokersArgsForCall = append(fake.getBrokersArgsForCall, struct{}{})
	fake.recordInvocation("GetBrokers", []interface{}{})
	fake.getBrokersMutex.Unlock()
	if fake.GetBrokersStub != nil {
		return fake.GetBrokersStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.getBrokersReturns.result1, fake.getBrokersReturns.result2
}

func (fake *FakeClient) GetBrokersCallCount() int {
	fake.getBrokersMutex.RLock()
	defer fake.getBrokersMutex.RUnlock()
	return len(fake.getBrokersArgsForCall)
}

func (fake *FakeClient) GetBrokersReturns(result1 []sm.Broker, result2 error) {
	fake.GetBrokersStub = nil
	fake.getBrokersReturns = struct {
		result1 []sm.Broker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) GetBrokersReturnsOnCall(i int, result1 []sm.Broker, result2 error) {
	fake.GetBrokersStub = nil
	if fake.getBrokersReturnsOnCall == nil {
		fake.getBrokersReturnsOnCall = make(map[int]struct {
			result1 []sm.Broker
			result2 error
		})
	}
	fake.getBrokersReturnsOnCall[i] = struct {
		result1 []sm.Broker
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getBrokersMutex.RLock()
	defer fake.getBrokersMutex.RUnlock()
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
