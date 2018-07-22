package mocks

import (
	"context"
	"net/http"

	"flamingo.me/flamingo/framework/router"
	"flamingo.me/flamingo/framework/web"
	"github.com/stretchr/testify/mock"
)

// Filter is an autogenerated mock type for the Filter type
type Filter struct {
	mock.Mock
}

// Filter provides a mock function with given fields: ctx, w, fc
func (_m *Filter) Filter(ctx context.Context, r *web.Request, w http.ResponseWriter, fc *router.FilterChain) web.Response {
	ret := _m.Called(ctx, w, fc)

	var r0 web.Response
	if rf, ok := ret.Get(0).(func(context.Context, *web.Request, http.ResponseWriter, *router.FilterChain) web.Response); ok {
		r0 = rf(ctx, r, w, fc)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(web.Response)
		}
	}

	return r0
}
