// Code generated by mockery. DO NOT EDIT.

package translation

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	datafederation "github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/translation/datafederation"
)

// DatafederationPrivateEndpointServiceMock is an autogenerated mock type for the DatafederationPrivateEndpointService type
type DatafederationPrivateEndpointServiceMock struct {
	mock.Mock
}

type DatafederationPrivateEndpointServiceMock_Expecter struct {
	mock *mock.Mock
}

func (_m *DatafederationPrivateEndpointServiceMock) EXPECT() *DatafederationPrivateEndpointServiceMock_Expecter {
	return &DatafederationPrivateEndpointServiceMock_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: _a0, _a1
func (_m *DatafederationPrivateEndpointServiceMock) Create(_a0 context.Context, _a1 *datafederation.DatafederationPrivateEndpointEntry) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *datafederation.DatafederationPrivateEndpointEntry) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DatafederationPrivateEndpointServiceMock_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type DatafederationPrivateEndpointServiceMock_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *datafederation.DatafederationPrivateEndpointEntry
func (_e *DatafederationPrivateEndpointServiceMock_Expecter) Create(_a0 interface{}, _a1 interface{}) *DatafederationPrivateEndpointServiceMock_Create_Call {
	return &DatafederationPrivateEndpointServiceMock_Create_Call{Call: _e.mock.On("Create", _a0, _a1)}
}

func (_c *DatafederationPrivateEndpointServiceMock_Create_Call) Run(run func(_a0 context.Context, _a1 *datafederation.DatafederationPrivateEndpointEntry)) *DatafederationPrivateEndpointServiceMock_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*datafederation.DatafederationPrivateEndpointEntry))
	})
	return _c
}

func (_c *DatafederationPrivateEndpointServiceMock_Create_Call) Return(_a0 error) *DatafederationPrivateEndpointServiceMock_Create_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DatafederationPrivateEndpointServiceMock_Create_Call) RunAndReturn(run func(context.Context, *datafederation.DatafederationPrivateEndpointEntry) error) *DatafederationPrivateEndpointServiceMock_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: _a0, _a1
func (_m *DatafederationPrivateEndpointServiceMock) Delete(_a0 context.Context, _a1 *datafederation.DatafederationPrivateEndpointEntry) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *datafederation.DatafederationPrivateEndpointEntry) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DatafederationPrivateEndpointServiceMock_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type DatafederationPrivateEndpointServiceMock_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *datafederation.DatafederationPrivateEndpointEntry
func (_e *DatafederationPrivateEndpointServiceMock_Expecter) Delete(_a0 interface{}, _a1 interface{}) *DatafederationPrivateEndpointServiceMock_Delete_Call {
	return &DatafederationPrivateEndpointServiceMock_Delete_Call{Call: _e.mock.On("Delete", _a0, _a1)}
}

func (_c *DatafederationPrivateEndpointServiceMock_Delete_Call) Run(run func(_a0 context.Context, _a1 *datafederation.DatafederationPrivateEndpointEntry)) *DatafederationPrivateEndpointServiceMock_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*datafederation.DatafederationPrivateEndpointEntry))
	})
	return _c
}

func (_c *DatafederationPrivateEndpointServiceMock_Delete_Call) Return(_a0 error) *DatafederationPrivateEndpointServiceMock_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DatafederationPrivateEndpointServiceMock_Delete_Call) RunAndReturn(run func(context.Context, *datafederation.DatafederationPrivateEndpointEntry) error) *DatafederationPrivateEndpointServiceMock_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, projectID
func (_m *DatafederationPrivateEndpointServiceMock) List(ctx context.Context, projectID string) ([]*datafederation.DatafederationPrivateEndpointEntry, error) {
	ret := _m.Called(ctx, projectID)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 []*datafederation.DatafederationPrivateEndpointEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*datafederation.DatafederationPrivateEndpointEntry, error)); ok {
		return rf(ctx, projectID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []*datafederation.DatafederationPrivateEndpointEntry); ok {
		r0 = rf(ctx, projectID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*datafederation.DatafederationPrivateEndpointEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, projectID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DatafederationPrivateEndpointServiceMock_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type DatafederationPrivateEndpointServiceMock_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - projectID string
func (_e *DatafederationPrivateEndpointServiceMock_Expecter) List(ctx interface{}, projectID interface{}) *DatafederationPrivateEndpointServiceMock_List_Call {
	return &DatafederationPrivateEndpointServiceMock_List_Call{Call: _e.mock.On("List", ctx, projectID)}
}

func (_c *DatafederationPrivateEndpointServiceMock_List_Call) Run(run func(ctx context.Context, projectID string)) *DatafederationPrivateEndpointServiceMock_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *DatafederationPrivateEndpointServiceMock_List_Call) Return(_a0 []*datafederation.DatafederationPrivateEndpointEntry, _a1 error) *DatafederationPrivateEndpointServiceMock_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DatafederationPrivateEndpointServiceMock_List_Call) RunAndReturn(run func(context.Context, string) ([]*datafederation.DatafederationPrivateEndpointEntry, error)) *DatafederationPrivateEndpointServiceMock_List_Call {
	_c.Call.Return(run)
	return _c
}

// NewDatafederationPrivateEndpointServiceMock creates a new instance of DatafederationPrivateEndpointServiceMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDatafederationPrivateEndpointServiceMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *DatafederationPrivateEndpointServiceMock {
	mock := &DatafederationPrivateEndpointServiceMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}