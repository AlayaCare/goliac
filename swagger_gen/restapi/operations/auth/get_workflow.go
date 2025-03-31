// Code generated by go-swagger; DO NOT EDIT.

package auth

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// GetWorkflowHandlerFunc turns a function with the right signature into a get workflow handler
type GetWorkflowHandlerFunc func(GetWorkflowParams) middleware.Responder

// Handle executing the request and returning a response
func (fn GetWorkflowHandlerFunc) Handle(params GetWorkflowParams) middleware.Responder {
	return fn(params)
}

// GetWorkflowHandler interface for that can handle valid get workflow params
type GetWorkflowHandler interface {
	Handle(GetWorkflowParams) middleware.Responder
}

// NewGetWorkflow creates a new http.Handler for the get workflow operation
func NewGetWorkflow(ctx *middleware.Context, handler GetWorkflowHandler) *GetWorkflow {
	return &GetWorkflow{Context: ctx, Handler: handler}
}

/*
	GetWorkflow swagger:route GET /auth/workflows/{workflowName} auth getWorkflow

Get Workflow information
*/
type GetWorkflow struct {
	Context *middleware.Context
	Handler GetWorkflowHandler
}

func (o *GetWorkflow) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetWorkflowParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}

// GetWorkflowOKBody get workflow o k body
//
// swagger:model GetWorkflowOKBody
type GetWorkflowOKBody struct {

	// workflow type
	// Min Length: 1
	WorkflowType string `json:"workflow_type,omitempty"`
}

// Validate validates this get workflow o k body
func (o *GetWorkflowOKBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateWorkflowType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetWorkflowOKBody) validateWorkflowType(formats strfmt.Registry) error {
	if swag.IsZero(o.WorkflowType) { // not required
		return nil
	}

	if err := validate.MinLength("getWorkflowOK"+"."+"workflow_type", "body", o.WorkflowType, 1); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this get workflow o k body based on context it is used
func (o *GetWorkflowOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *GetWorkflowOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetWorkflowOKBody) UnmarshalBinary(b []byte) error {
	var res GetWorkflowOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
