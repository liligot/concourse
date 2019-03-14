// Code generated by counterfeiter. DO NOT EDIT.
package resourcefakes

import (
	context "context"
	sync "sync"

	lager "code.cloudfoundry.org/lager"
	atc "github.com/concourse/concourse/atc"
	creds "github.com/concourse/concourse/atc/creds"
	resource "github.com/concourse/concourse/atc/resource"
	v2 "github.com/concourse/concourse/atc/resource/v2"
	worker "github.com/concourse/concourse/atc/worker"
)

type FakeFetcher struct {
	FetchStub        func(context.Context, lager.Logger, resource.Session, v2.GetEventHandler, atc.Tags, int, creds.VersionedResourceTypes, resource.ResourceInstance, resource.Metadata, worker.ImageFetchingDelegate) (worker.Volume, error)
	fetchMutex       sync.RWMutex
	fetchArgsForCall []struct {
		arg1  context.Context
		arg2  lager.Logger
		arg3  resource.Session
		arg4  v2.GetEventHandler
		arg5  atc.Tags
		arg6  int
		arg7  creds.VersionedResourceTypes
		arg8  resource.ResourceInstance
		arg9  resource.Metadata
		arg10 worker.ImageFetchingDelegate
	}
	fetchReturns struct {
		result1 worker.Volume
		result2 error
	}
	fetchReturnsOnCall map[int]struct {
		result1 worker.Volume
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeFetcher) Fetch(arg1 context.Context, arg2 lager.Logger, arg3 resource.Session, arg4 v2.GetEventHandler, arg5 atc.Tags, arg6 int, arg7 creds.VersionedResourceTypes, arg8 resource.ResourceInstance, arg9 resource.Metadata, arg10 worker.ImageFetchingDelegate) (worker.Volume, error) {
	fake.fetchMutex.Lock()
	ret, specificReturn := fake.fetchReturnsOnCall[len(fake.fetchArgsForCall)]
	fake.fetchArgsForCall = append(fake.fetchArgsForCall, struct {
		arg1  context.Context
		arg2  lager.Logger
		arg3  resource.Session
		arg4  v2.GetEventHandler
		arg5  atc.Tags
		arg6  int
		arg7  creds.VersionedResourceTypes
		arg8  resource.ResourceInstance
		arg9  resource.Metadata
		arg10 worker.ImageFetchingDelegate
	}{arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10})
	fake.recordInvocation("Fetch", []interface{}{arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10})
	fake.fetchMutex.Unlock()
	if fake.FetchStub != nil {
		return fake.FetchStub(arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.fetchReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeFetcher) FetchCallCount() int {
	fake.fetchMutex.RLock()
	defer fake.fetchMutex.RUnlock()
	return len(fake.fetchArgsForCall)
}

func (fake *FakeFetcher) FetchCalls(stub func(context.Context, lager.Logger, resource.Session, v2.GetEventHandler, atc.Tags, int, creds.VersionedResourceTypes, resource.ResourceInstance, resource.Metadata, worker.ImageFetchingDelegate) (worker.Volume, error)) {
	fake.fetchMutex.Lock()
	defer fake.fetchMutex.Unlock()
	fake.FetchStub = stub
}

func (fake *FakeFetcher) FetchArgsForCall(i int) (context.Context, lager.Logger, resource.Session, v2.GetEventHandler, atc.Tags, int, creds.VersionedResourceTypes, resource.ResourceInstance, resource.Metadata, worker.ImageFetchingDelegate) {
	fake.fetchMutex.RLock()
	defer fake.fetchMutex.RUnlock()
	argsForCall := fake.fetchArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5, argsForCall.arg6, argsForCall.arg7, argsForCall.arg8, argsForCall.arg9, argsForCall.arg10
}

func (fake *FakeFetcher) FetchReturns(result1 worker.Volume, result2 error) {
	fake.fetchMutex.Lock()
	defer fake.fetchMutex.Unlock()
	fake.FetchStub = nil
	fake.fetchReturns = struct {
		result1 worker.Volume
		result2 error
	}{result1, result2}
}

func (fake *FakeFetcher) FetchReturnsOnCall(i int, result1 worker.Volume, result2 error) {
	fake.fetchMutex.Lock()
	defer fake.fetchMutex.Unlock()
	fake.FetchStub = nil
	if fake.fetchReturnsOnCall == nil {
		fake.fetchReturnsOnCall = make(map[int]struct {
			result1 worker.Volume
			result2 error
		})
	}
	fake.fetchReturnsOnCall[i] = struct {
		result1 worker.Volume
		result2 error
	}{result1, result2}
}

func (fake *FakeFetcher) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.fetchMutex.RLock()
	defer fake.fetchMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeFetcher) recordInvocation(key string, args []interface{}) {
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

var _ resource.Fetcher = new(FakeFetcher)
