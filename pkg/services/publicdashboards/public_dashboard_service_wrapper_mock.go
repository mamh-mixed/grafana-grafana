// Code generated by mockery v2.16.0. DO NOT EDIT.

package publicdashboards

import (
	context "context"

	models "github.com/grafana/grafana/pkg/services/publicdashboards/models"
	mock "github.com/stretchr/testify/mock"
)

// FakePublicDashboardServiceWrapper is an autogenerated mock type for the ServiceWrapper type
type FakePublicDashboardServiceWrapper struct {
	mock.Mock
}

// Delete provides a mock function with given fields: ctx, uid
func (_m *FakePublicDashboardServiceWrapper) Delete(ctx context.Context, uid string) error {
	ret := _m.Called(ctx, uid)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, uid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FindByDashboardUid provides a mock function with given fields: ctx, orgId, dashboardUid
func (_m *FakePublicDashboardServiceWrapper) FindByDashboardUid(ctx context.Context, orgId int64, dashboardUid string) (*models.PublicDashboard, error) {
	ret := _m.Called(ctx, orgId, dashboardUid)

	var r0 *models.PublicDashboard
	if rf, ok := ret.Get(0).(func(context.Context, int64, string) *models.PublicDashboard); ok {
		r0 = rf(ctx, orgId, dashboardUid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.PublicDashboard)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64, string) error); ok {
		r1 = rf(ctx, orgId, dashboardUid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewFakePublicDashboardServiceWrapper interface {
	mock.TestingT
	Cleanup(func())
}

// NewFakePublicDashboardServiceWrapper creates a new instance of FakePublicDashboardServiceWrapper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFakePublicDashboardServiceWrapper(t mockConstructorTestingTNewFakePublicDashboardServiceWrapper) *FakePublicDashboardServiceWrapper {
	mock := &FakePublicDashboardServiceWrapper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
