// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	context "context"

	api "github.com/DataDog/datadog-agent/pkg/security/proto/api"

	mock "github.com/stretchr/testify/mock"
)

// SecurityModuleServer is an autogenerated mock type for the SecurityModuleServer type
type SecurityModuleServer struct {
	mock.Mock
}

// DumpActivity provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) DumpActivity(_a0 context.Context, _a1 *api.ActivityDumpParams) (*api.ActivityDumpMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.ActivityDumpMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.ActivityDumpParams) (*api.ActivityDumpMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.ActivityDumpParams) *api.ActivityDumpMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ActivityDumpMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.ActivityDumpParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DumpDiscarders provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) DumpDiscarders(_a0 context.Context, _a1 *api.DumpDiscardersParams) (*api.DumpDiscardersMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.DumpDiscardersMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.DumpDiscardersParams) (*api.DumpDiscardersMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.DumpDiscardersParams) *api.DumpDiscardersMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.DumpDiscardersMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.DumpDiscardersParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DumpNetworkNamespace provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) DumpNetworkNamespace(_a0 context.Context, _a1 *api.DumpNetworkNamespaceParams) (*api.DumpNetworkNamespaceMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.DumpNetworkNamespaceMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.DumpNetworkNamespaceParams) (*api.DumpNetworkNamespaceMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.DumpNetworkNamespaceParams) *api.DumpNetworkNamespaceMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.DumpNetworkNamespaceMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.DumpNetworkNamespaceParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DumpProcessCache provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) DumpProcessCache(_a0 context.Context, _a1 *api.DumpProcessCacheParams) (*api.SecurityDumpProcessCacheMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.SecurityDumpProcessCacheMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.DumpProcessCacheParams) (*api.SecurityDumpProcessCacheMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.DumpProcessCacheParams) *api.SecurityDumpProcessCacheMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.SecurityDumpProcessCacheMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.DumpProcessCacheParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetActivityDumpStream provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) GetActivityDumpStream(_a0 *api.ActivityDumpStreamParams, _a1 api.SecurityModule_GetActivityDumpStreamServer) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*api.ActivityDumpStreamParams, api.SecurityModule_GetActivityDumpStreamServer) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetConfig provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) GetConfig(_a0 context.Context, _a1 *api.GetConfigParams) (*api.SecurityConfigMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.SecurityConfigMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.GetConfigParams) (*api.SecurityConfigMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.GetConfigParams) *api.SecurityConfigMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.SecurityConfigMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.GetConfigParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEvents provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) GetEvents(_a0 *api.GetEventParams, _a1 api.SecurityModule_GetEventsServer) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*api.GetEventParams, api.SecurityModule_GetEventsServer) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetStatus provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) GetStatus(_a0 context.Context, _a1 *api.GetStatusParams) (*api.Status, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.Status
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.GetStatusParams) (*api.Status, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.GetStatusParams) *api.Status); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.Status)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.GetStatusParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListActivityDumps provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) ListActivityDumps(_a0 context.Context, _a1 *api.ActivityDumpListParams) (*api.ActivityDumpListMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.ActivityDumpListMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.ActivityDumpListParams) (*api.ActivityDumpListMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.ActivityDumpListParams) *api.ActivityDumpListMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ActivityDumpListMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.ActivityDumpListParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListSecurityProfiles provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) ListSecurityProfiles(_a0 context.Context, _a1 *api.SecurityProfileListParams) (*api.SecurityProfileListMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.SecurityProfileListMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.SecurityProfileListParams) (*api.SecurityProfileListMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.SecurityProfileListParams) *api.SecurityProfileListMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.SecurityProfileListMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.SecurityProfileListParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReloadPolicies provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) ReloadPolicies(_a0 context.Context, _a1 *api.ReloadPoliciesParams) (*api.ReloadPoliciesResultMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.ReloadPoliciesResultMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.ReloadPoliciesParams) (*api.ReloadPoliciesResultMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.ReloadPoliciesParams) *api.ReloadPoliciesResultMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ReloadPoliciesResultMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.ReloadPoliciesParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RunSelfTest provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) RunSelfTest(_a0 context.Context, _a1 *api.RunSelfTestParams) (*api.SecuritySelfTestResultMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.SecuritySelfTestResultMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.RunSelfTestParams) (*api.SecuritySelfTestResultMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.RunSelfTestParams) *api.SecuritySelfTestResultMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.SecuritySelfTestResultMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.RunSelfTestParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveSecurityProfile provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) SaveSecurityProfile(_a0 context.Context, _a1 *api.SecurityProfileSaveParams) (*api.SecurityProfileSaveMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.SecurityProfileSaveMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.SecurityProfileSaveParams) (*api.SecurityProfileSaveMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.SecurityProfileSaveParams) *api.SecurityProfileSaveMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.SecurityProfileSaveMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.SecurityProfileSaveParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StopActivityDump provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) StopActivityDump(_a0 context.Context, _a1 *api.ActivityDumpStopParams) (*api.ActivityDumpStopMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.ActivityDumpStopMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.ActivityDumpStopParams) (*api.ActivityDumpStopMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.ActivityDumpStopParams) *api.ActivityDumpStopMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ActivityDumpStopMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.ActivityDumpStopParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TranscodingRequest provides a mock function with given fields: _a0, _a1
func (_m *SecurityModuleServer) TranscodingRequest(_a0 context.Context, _a1 *api.TranscodingRequestParams) (*api.TranscodingRequestMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *api.TranscodingRequestMessage
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.TranscodingRequestParams) (*api.TranscodingRequestMessage, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *api.TranscodingRequestParams) *api.TranscodingRequestMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.TranscodingRequestMessage)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *api.TranscodingRequestParams) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mustEmbedUnimplementedSecurityModuleServer provides a mock function with given fields:
func (_m *SecurityModuleServer) mustEmbedUnimplementedSecurityModuleServer() {
	_m.Called()
}

type mockConstructorTestingTNewSecurityModuleServer interface {
	mock.TestingT
	Cleanup(func())
}

// NewSecurityModuleServer creates a new instance of SecurityModuleServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSecurityModuleServer(t mockConstructorTestingTNewSecurityModuleServer) *SecurityModuleServer {
	mock := &SecurityModuleServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
