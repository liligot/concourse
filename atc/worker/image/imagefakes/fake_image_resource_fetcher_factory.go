// Code generated by counterfeiter. DO NOT EDIT.
package imagefakes

import (
	sync "sync"

	atc "github.com/concourse/concourse/atc"
	creds "github.com/concourse/concourse/atc/creds"
	resource "github.com/concourse/concourse/atc/resource"
	worker "github.com/concourse/concourse/atc/worker"
	image "github.com/concourse/concourse/atc/worker/image"
)

type FakeImageResourceFetcherFactory struct {
	NewImageResourceFetcherStub        func(worker.Worker, resource.ResourceFactory, worker.ImageResource, atc.Version, atc.Space, int, creds.VersionedResourceTypes, worker.ImageFetchingDelegate) image.ImageResourceFetcher
	newImageResourceFetcherMutex       sync.RWMutex
	newImageResourceFetcherArgsForCall []struct {
		arg1 worker.Worker
		arg2 resource.ResourceFactory
		arg3 worker.ImageResource
		arg4 atc.Version
		arg5 atc.Space
		arg6 int
		arg7 creds.VersionedResourceTypes
		arg8 worker.ImageFetchingDelegate
	}
	newImageResourceFetcherReturns struct {
		result1 image.ImageResourceFetcher
	}
	newImageResourceFetcherReturnsOnCall map[int]struct {
		result1 image.ImageResourceFetcher
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeImageResourceFetcherFactory) NewImageResourceFetcher(arg1 worker.Worker, arg2 resource.ResourceFactory, arg3 worker.ImageResource, arg4 atc.Version, arg5 atc.Space, arg6 int, arg7 creds.VersionedResourceTypes, arg8 worker.ImageFetchingDelegate) image.ImageResourceFetcher {
	fake.newImageResourceFetcherMutex.Lock()
	ret, specificReturn := fake.newImageResourceFetcherReturnsOnCall[len(fake.newImageResourceFetcherArgsForCall)]
	fake.newImageResourceFetcherArgsForCall = append(fake.newImageResourceFetcherArgsForCall, struct {
		arg1 worker.Worker
		arg2 resource.ResourceFactory
		arg3 worker.ImageResource
		arg4 atc.Version
		arg5 atc.Space
		arg6 int
		arg7 creds.VersionedResourceTypes
		arg8 worker.ImageFetchingDelegate
	}{arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8})
	fake.recordInvocation("NewImageResourceFetcher", []interface{}{arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8})
	fake.newImageResourceFetcherMutex.Unlock()
	if fake.NewImageResourceFetcherStub != nil {
		return fake.NewImageResourceFetcherStub(arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.newImageResourceFetcherReturns
	return fakeReturns.result1
}

func (fake *FakeImageResourceFetcherFactory) NewImageResourceFetcherCallCount() int {
	fake.newImageResourceFetcherMutex.RLock()
	defer fake.newImageResourceFetcherMutex.RUnlock()
	return len(fake.newImageResourceFetcherArgsForCall)
}

func (fake *FakeImageResourceFetcherFactory) NewImageResourceFetcherCalls(stub func(worker.Worker, resource.ResourceFactory, worker.ImageResource, atc.Version, atc.Space, int, creds.VersionedResourceTypes, worker.ImageFetchingDelegate) image.ImageResourceFetcher) {
	fake.newImageResourceFetcherMutex.Lock()
	defer fake.newImageResourceFetcherMutex.Unlock()
	fake.NewImageResourceFetcherStub = stub
}

func (fake *FakeImageResourceFetcherFactory) NewImageResourceFetcherArgsForCall(i int) (worker.Worker, resource.ResourceFactory, worker.ImageResource, atc.Version, atc.Space, int, creds.VersionedResourceTypes, worker.ImageFetchingDelegate) {
	fake.newImageResourceFetcherMutex.RLock()
	defer fake.newImageResourceFetcherMutex.RUnlock()
	argsForCall := fake.newImageResourceFetcherArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5, argsForCall.arg6, argsForCall.arg7, argsForCall.arg8
}

func (fake *FakeImageResourceFetcherFactory) NewImageResourceFetcherReturns(result1 image.ImageResourceFetcher) {
	fake.newImageResourceFetcherMutex.Lock()
	defer fake.newImageResourceFetcherMutex.Unlock()
	fake.NewImageResourceFetcherStub = nil
	fake.newImageResourceFetcherReturns = struct {
		result1 image.ImageResourceFetcher
	}{result1}
}

func (fake *FakeImageResourceFetcherFactory) NewImageResourceFetcherReturnsOnCall(i int, result1 image.ImageResourceFetcher) {
	fake.newImageResourceFetcherMutex.Lock()
	defer fake.newImageResourceFetcherMutex.Unlock()
	fake.NewImageResourceFetcherStub = nil
	if fake.newImageResourceFetcherReturnsOnCall == nil {
		fake.newImageResourceFetcherReturnsOnCall = make(map[int]struct {
			result1 image.ImageResourceFetcher
		})
	}
	fake.newImageResourceFetcherReturnsOnCall[i] = struct {
		result1 image.ImageResourceFetcher
	}{result1}
}

func (fake *FakeImageResourceFetcherFactory) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.newImageResourceFetcherMutex.RLock()
	defer fake.newImageResourceFetcherMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeImageResourceFetcherFactory) recordInvocation(key string, args []interface{}) {
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

var _ image.ImageResourceFetcherFactory = new(FakeImageResourceFetcherFactory)
