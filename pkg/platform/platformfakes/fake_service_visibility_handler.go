// Code generated by counterfeiter. DO NOT EDIT.
package platformfakes

import (
	context "context"
	json "encoding/json"
	sync "sync"

	platform "github.com/Peripli/service-broker-proxy/pkg/platform"
	types "github.com/Peripli/service-manager/pkg/types"
)

type FakeServiceVisibilityHandler struct {
	ConvertStub        func(*types.Visibility, *types.ServicePlan) []*platform.ServiceVisibilityEntity
	convertMutex       sync.RWMutex
	convertArgsForCall []struct {
		arg1 *types.Visibility
		arg2 *types.ServicePlan
	}
	convertReturns struct {
		result1 []*platform.ServiceVisibilityEntity
	}
	convertReturnsOnCall map[int]struct {
		result1 []*platform.ServiceVisibilityEntity
	}
	DisableAccessForPlanStub        func(context.Context, json.RawMessage, string) error
	disableAccessForPlanMutex       sync.RWMutex
	disableAccessForPlanArgsForCall []struct {
		arg1 context.Context
		arg2 json.RawMessage
		arg3 string
	}
	disableAccessForPlanReturns struct {
		result1 error
	}
	disableAccessForPlanReturnsOnCall map[int]struct {
		result1 error
	}
	EnableAccessForPlanStub        func(context.Context, json.RawMessage, string) error
	enableAccessForPlanMutex       sync.RWMutex
	enableAccessForPlanArgsForCall []struct {
		arg1 context.Context
		arg2 json.RawMessage
		arg3 string
	}
	enableAccessForPlanReturns struct {
		result1 error
	}
	enableAccessForPlanReturnsOnCall map[int]struct {
		result1 error
	}
	GetVisibilitiesByPlansStub        func(context.Context, []*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error)
	getVisibilitiesByPlansMutex       sync.RWMutex
	getVisibilitiesByPlansArgsForCall []struct {
		arg1 context.Context
		arg2 []*types.ServicePlan
	}
	getVisibilitiesByPlansReturns struct {
		result1 []*platform.ServiceVisibilityEntity
		result2 error
	}
	getVisibilitiesByPlansReturnsOnCall map[int]struct {
		result1 []*platform.ServiceVisibilityEntity
		result2 error
	}
	MapStub        func(*platform.ServiceVisibilityEntity) string
	mapMutex       sync.RWMutex
	mapArgsForCall []struct {
		arg1 *platform.ServiceVisibilityEntity
	}
	mapReturns struct {
		result1 string
	}
	mapReturnsOnCall map[int]struct {
		result1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeServiceVisibilityHandler) Convert(arg1 *types.Visibility, arg2 *types.ServicePlan) []*platform.ServiceVisibilityEntity {
	fake.convertMutex.Lock()
	ret, specificReturn := fake.convertReturnsOnCall[len(fake.convertArgsForCall)]
	fake.convertArgsForCall = append(fake.convertArgsForCall, struct {
		arg1 *types.Visibility
		arg2 *types.ServicePlan
	}{arg1, arg2})
	fake.recordInvocation("Convert", []interface{}{arg1, arg2})
	fake.convertMutex.Unlock()
	if fake.ConvertStub != nil {
		return fake.ConvertStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.convertReturns
	return fakeReturns.result1
}

func (fake *FakeServiceVisibilityHandler) ConvertCallCount() int {
	fake.convertMutex.RLock()
	defer fake.convertMutex.RUnlock()
	return len(fake.convertArgsForCall)
}

func (fake *FakeServiceVisibilityHandler) ConvertCalls(stub func(*types.Visibility, *types.ServicePlan) []*platform.ServiceVisibilityEntity) {
	fake.convertMutex.Lock()
	defer fake.convertMutex.Unlock()
	fake.ConvertStub = stub
}

func (fake *FakeServiceVisibilityHandler) ConvertArgsForCall(i int) (*types.Visibility, *types.ServicePlan) {
	fake.convertMutex.RLock()
	defer fake.convertMutex.RUnlock()
	argsForCall := fake.convertArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceVisibilityHandler) ConvertReturns(result1 []*platform.ServiceVisibilityEntity) {
	fake.convertMutex.Lock()
	defer fake.convertMutex.Unlock()
	fake.ConvertStub = nil
	fake.convertReturns = struct {
		result1 []*platform.ServiceVisibilityEntity
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) ConvertReturnsOnCall(i int, result1 []*platform.ServiceVisibilityEntity) {
	fake.convertMutex.Lock()
	defer fake.convertMutex.Unlock()
	fake.ConvertStub = nil
	if fake.convertReturnsOnCall == nil {
		fake.convertReturnsOnCall = make(map[int]struct {
			result1 []*platform.ServiceVisibilityEntity
		})
	}
	fake.convertReturnsOnCall[i] = struct {
		result1 []*platform.ServiceVisibilityEntity
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) DisableAccessForPlan(arg1 context.Context, arg2 json.RawMessage, arg3 string) error {
	fake.disableAccessForPlanMutex.Lock()
	ret, specificReturn := fake.disableAccessForPlanReturnsOnCall[len(fake.disableAccessForPlanArgsForCall)]
	fake.disableAccessForPlanArgsForCall = append(fake.disableAccessForPlanArgsForCall, struct {
		arg1 context.Context
		arg2 json.RawMessage
		arg3 string
	}{arg1, arg2, arg3})
	fake.recordInvocation("DisableAccessForPlan", []interface{}{arg1, arg2, arg3})
	fake.disableAccessForPlanMutex.Unlock()
	if fake.DisableAccessForPlanStub != nil {
		return fake.DisableAccessForPlanStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.disableAccessForPlanReturns
	return fakeReturns.result1
}

func (fake *FakeServiceVisibilityHandler) DisableAccessForPlanCallCount() int {
	fake.disableAccessForPlanMutex.RLock()
	defer fake.disableAccessForPlanMutex.RUnlock()
	return len(fake.disableAccessForPlanArgsForCall)
}

func (fake *FakeServiceVisibilityHandler) DisableAccessForPlanCalls(stub func(context.Context, json.RawMessage, string) error) {
	fake.disableAccessForPlanMutex.Lock()
	defer fake.disableAccessForPlanMutex.Unlock()
	fake.DisableAccessForPlanStub = stub
}

func (fake *FakeServiceVisibilityHandler) DisableAccessForPlanArgsForCall(i int) (context.Context, json.RawMessage, string) {
	fake.disableAccessForPlanMutex.RLock()
	defer fake.disableAccessForPlanMutex.RUnlock()
	argsForCall := fake.disableAccessForPlanArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeServiceVisibilityHandler) DisableAccessForPlanReturns(result1 error) {
	fake.disableAccessForPlanMutex.Lock()
	defer fake.disableAccessForPlanMutex.Unlock()
	fake.DisableAccessForPlanStub = nil
	fake.disableAccessForPlanReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) DisableAccessForPlanReturnsOnCall(i int, result1 error) {
	fake.disableAccessForPlanMutex.Lock()
	defer fake.disableAccessForPlanMutex.Unlock()
	fake.DisableAccessForPlanStub = nil
	if fake.disableAccessForPlanReturnsOnCall == nil {
		fake.disableAccessForPlanReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.disableAccessForPlanReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) EnableAccessForPlan(arg1 context.Context, arg2 json.RawMessage, arg3 string) error {
	fake.enableAccessForPlanMutex.Lock()
	ret, specificReturn := fake.enableAccessForPlanReturnsOnCall[len(fake.enableAccessForPlanArgsForCall)]
	fake.enableAccessForPlanArgsForCall = append(fake.enableAccessForPlanArgsForCall, struct {
		arg1 context.Context
		arg2 json.RawMessage
		arg3 string
	}{arg1, arg2, arg3})
	fake.recordInvocation("EnableAccessForPlan", []interface{}{arg1, arg2, arg3})
	fake.enableAccessForPlanMutex.Unlock()
	if fake.EnableAccessForPlanStub != nil {
		return fake.EnableAccessForPlanStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.enableAccessForPlanReturns
	return fakeReturns.result1
}

func (fake *FakeServiceVisibilityHandler) EnableAccessForPlanCallCount() int {
	fake.enableAccessForPlanMutex.RLock()
	defer fake.enableAccessForPlanMutex.RUnlock()
	return len(fake.enableAccessForPlanArgsForCall)
}

func (fake *FakeServiceVisibilityHandler) EnableAccessForPlanCalls(stub func(context.Context, json.RawMessage, string) error) {
	fake.enableAccessForPlanMutex.Lock()
	defer fake.enableAccessForPlanMutex.Unlock()
	fake.EnableAccessForPlanStub = stub
}

func (fake *FakeServiceVisibilityHandler) EnableAccessForPlanArgsForCall(i int) (context.Context, json.RawMessage, string) {
	fake.enableAccessForPlanMutex.RLock()
	defer fake.enableAccessForPlanMutex.RUnlock()
	argsForCall := fake.enableAccessForPlanArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeServiceVisibilityHandler) EnableAccessForPlanReturns(result1 error) {
	fake.enableAccessForPlanMutex.Lock()
	defer fake.enableAccessForPlanMutex.Unlock()
	fake.EnableAccessForPlanStub = nil
	fake.enableAccessForPlanReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) EnableAccessForPlanReturnsOnCall(i int, result1 error) {
	fake.enableAccessForPlanMutex.Lock()
	defer fake.enableAccessForPlanMutex.Unlock()
	fake.EnableAccessForPlanStub = nil
	if fake.enableAccessForPlanReturnsOnCall == nil {
		fake.enableAccessForPlanReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.enableAccessForPlanReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) GetVisibilitiesByPlans(arg1 context.Context, arg2 []*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	var arg2Copy []*types.ServicePlan
	if arg2 != nil {
		arg2Copy = make([]*types.ServicePlan, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.getVisibilitiesByPlansMutex.Lock()
	ret, specificReturn := fake.getVisibilitiesByPlansReturnsOnCall[len(fake.getVisibilitiesByPlansArgsForCall)]
	fake.getVisibilitiesByPlansArgsForCall = append(fake.getVisibilitiesByPlansArgsForCall, struct {
		arg1 context.Context
		arg2 []*types.ServicePlan
	}{arg1, arg2Copy})
	fake.recordInvocation("GetVisibilitiesByPlans", []interface{}{arg1, arg2Copy})
	fake.getVisibilitiesByPlansMutex.Unlock()
	if fake.GetVisibilitiesByPlansStub != nil {
		return fake.GetVisibilitiesByPlansStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getVisibilitiesByPlansReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeServiceVisibilityHandler) GetVisibilitiesByPlansCallCount() int {
	fake.getVisibilitiesByPlansMutex.RLock()
	defer fake.getVisibilitiesByPlansMutex.RUnlock()
	return len(fake.getVisibilitiesByPlansArgsForCall)
}

func (fake *FakeServiceVisibilityHandler) GetVisibilitiesByPlansCalls(stub func(context.Context, []*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error)) {
	fake.getVisibilitiesByPlansMutex.Lock()
	defer fake.getVisibilitiesByPlansMutex.Unlock()
	fake.GetVisibilitiesByPlansStub = stub
}

func (fake *FakeServiceVisibilityHandler) GetVisibilitiesByPlansArgsForCall(i int) (context.Context, []*types.ServicePlan) {
	fake.getVisibilitiesByPlansMutex.RLock()
	defer fake.getVisibilitiesByPlansMutex.RUnlock()
	argsForCall := fake.getVisibilitiesByPlansArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeServiceVisibilityHandler) GetVisibilitiesByPlansReturns(result1 []*platform.ServiceVisibilityEntity, result2 error) {
	fake.getVisibilitiesByPlansMutex.Lock()
	defer fake.getVisibilitiesByPlansMutex.Unlock()
	fake.GetVisibilitiesByPlansStub = nil
	fake.getVisibilitiesByPlansReturns = struct {
		result1 []*platform.ServiceVisibilityEntity
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceVisibilityHandler) GetVisibilitiesByPlansReturnsOnCall(i int, result1 []*platform.ServiceVisibilityEntity, result2 error) {
	fake.getVisibilitiesByPlansMutex.Lock()
	defer fake.getVisibilitiesByPlansMutex.Unlock()
	fake.GetVisibilitiesByPlansStub = nil
	if fake.getVisibilitiesByPlansReturnsOnCall == nil {
		fake.getVisibilitiesByPlansReturnsOnCall = make(map[int]struct {
			result1 []*platform.ServiceVisibilityEntity
			result2 error
		})
	}
	fake.getVisibilitiesByPlansReturnsOnCall[i] = struct {
		result1 []*platform.ServiceVisibilityEntity
		result2 error
	}{result1, result2}
}

func (fake *FakeServiceVisibilityHandler) Map(arg1 *platform.ServiceVisibilityEntity) string {
	fake.mapMutex.Lock()
	ret, specificReturn := fake.mapReturnsOnCall[len(fake.mapArgsForCall)]
	fake.mapArgsForCall = append(fake.mapArgsForCall, struct {
		arg1 *platform.ServiceVisibilityEntity
	}{arg1})
	fake.recordInvocation("Map", []interface{}{arg1})
	fake.mapMutex.Unlock()
	if fake.MapStub != nil {
		return fake.MapStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.mapReturns
	return fakeReturns.result1
}

func (fake *FakeServiceVisibilityHandler) MapCallCount() int {
	fake.mapMutex.RLock()
	defer fake.mapMutex.RUnlock()
	return len(fake.mapArgsForCall)
}

func (fake *FakeServiceVisibilityHandler) MapCalls(stub func(*platform.ServiceVisibilityEntity) string) {
	fake.mapMutex.Lock()
	defer fake.mapMutex.Unlock()
	fake.MapStub = stub
}

func (fake *FakeServiceVisibilityHandler) MapArgsForCall(i int) *platform.ServiceVisibilityEntity {
	fake.mapMutex.RLock()
	defer fake.mapMutex.RUnlock()
	argsForCall := fake.mapArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeServiceVisibilityHandler) MapReturns(result1 string) {
	fake.mapMutex.Lock()
	defer fake.mapMutex.Unlock()
	fake.MapStub = nil
	fake.mapReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) MapReturnsOnCall(i int, result1 string) {
	fake.mapMutex.Lock()
	defer fake.mapMutex.Unlock()
	fake.MapStub = nil
	if fake.mapReturnsOnCall == nil {
		fake.mapReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.mapReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeServiceVisibilityHandler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.convertMutex.RLock()
	defer fake.convertMutex.RUnlock()
	fake.disableAccessForPlanMutex.RLock()
	defer fake.disableAccessForPlanMutex.RUnlock()
	fake.enableAccessForPlanMutex.RLock()
	defer fake.enableAccessForPlanMutex.RUnlock()
	fake.getVisibilitiesByPlansMutex.RLock()
	defer fake.getVisibilitiesByPlansMutex.RUnlock()
	fake.mapMutex.RLock()
	defer fake.mapMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeServiceVisibilityHandler) recordInvocation(key string, args []interface{}) {
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

var _ platform.ServiceVisibilityHandler = new(FakeServiceVisibilityHandler)
