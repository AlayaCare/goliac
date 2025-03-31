// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// Status status
//
// swagger:model status
type Status struct {

	// detailed errors
	DetailedErrors []string `json:"detailedErrors"`

	// detailed warnings
	DetailedWarnings []string `json:"detailedWarnings"`

	// last sync error
	LastSyncError string `json:"lastSyncError,omitempty"`

	// last sync time
	// Min Length: 1
	LastSyncTime string `json:"lastSyncTime,omitempty"`

	// nb repos
	NbRepos int64 `json:"nbRepos"`

	// nb teams
	NbTeams int64 `json:"nbTeams"`

	// nb users
	NbUsers int64 `json:"nbUsers"`

	// nb users external
	NbUsersExternal int64 `json:"nbUsersExternal"`

	// nb workflows
	NbWorkflows int64 `json:"nbWorkflows"`

	// version
	Version string `json:"version,omitempty"`
}

// Validate validates this status
func (m *Status) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLastSyncTime(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Status) validateLastSyncTime(formats strfmt.Registry) error {
	if swag.IsZero(m.LastSyncTime) { // not required
		return nil
	}

	if err := validate.MinLength("lastSyncTime", "body", m.LastSyncTime, 1); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this status based on context it is used
func (m *Status) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Status) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Status) UnmarshalBinary(b []byte) error {
	var res Status
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
