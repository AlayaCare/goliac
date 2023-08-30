// Code generated by go-swagger; DO NOT EDIT.

package health

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// GetReadinessHandlerFunc turns a function with the right signature into a get readiness handler
type GetReadinessHandlerFunc func(GetReadinessParams) middleware.Responder

// Handle executing the request and returning a response
func (fn GetReadinessHandlerFunc) Handle(params GetReadinessParams) middleware.Responder {
	return fn(params)
}

// GetReadinessHandler interface for that can handle valid get readiness params
type GetReadinessHandler interface {
	Handle(GetReadinessParams) middleware.Responder
}

// NewGetReadiness creates a new http.Handler for the get readiness operation
func NewGetReadiness(ctx *middleware.Context, handler GetReadinessHandler) *GetReadiness {
	return &GetReadiness{Context: ctx, Handler: handler}
}

/*
	GetReadiness swagger:route GET /readiness health getReadiness

Check if Goliac is ready to serve
*/
type GetReadiness struct {
	Context *middleware.Context
	Handler GetReadinessHandler
}

func (o *GetReadiness) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetReadinessParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}