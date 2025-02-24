// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// RepositoryDetails repository details
//
// swagger:model repositoryDetails
type RepositoryDetails struct {

	// allow update branch
	AllowUpdateBranch bool `json:"allowUpdateBranch"`

	// archived
	Archived bool `json:"archived"`

	// auto merge allowed
	AutoMergeAllowed bool `json:"autoMergeAllowed"`

	// collaborators
	Collaborators []*RepositoryDetailsCollaboratorsItems0 `json:"collaborators"`

	// delete branch on merge
	DeleteBranchOnMerge bool `json:"deleteBranchOnMerge"`

	// name
	Name string `json:"name,omitempty"`

	// teams
	Teams []*RepositoryDetailsTeamsItems0 `json:"teams"`

	// visibility
	Visibility string `json:"visibility"`
}

// Validate validates this repository details
func (m *RepositoryDetails) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCollaborators(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTeams(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *RepositoryDetails) validateCollaborators(formats strfmt.Registry) error {
	if swag.IsZero(m.Collaborators) { // not required
		return nil
	}

	for i := 0; i < len(m.Collaborators); i++ {
		if swag.IsZero(m.Collaborators[i]) { // not required
			continue
		}

		if m.Collaborators[i] != nil {
			if err := m.Collaborators[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("collaborators" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("collaborators" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *RepositoryDetails) validateTeams(formats strfmt.Registry) error {
	if swag.IsZero(m.Teams) { // not required
		return nil
	}

	for i := 0; i < len(m.Teams); i++ {
		if swag.IsZero(m.Teams[i]) { // not required
			continue
		}

		if m.Teams[i] != nil {
			if err := m.Teams[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("teams" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("teams" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this repository details based on the context it is used
func (m *RepositoryDetails) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCollaborators(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateTeams(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *RepositoryDetails) contextValidateCollaborators(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Collaborators); i++ {

		if m.Collaborators[i] != nil {

			if swag.IsZero(m.Collaborators[i]) { // not required
				return nil
			}

			if err := m.Collaborators[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("collaborators" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("collaborators" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *RepositoryDetails) contextValidateTeams(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Teams); i++ {

		if m.Teams[i] != nil {

			if swag.IsZero(m.Teams[i]) { // not required
				return nil
			}

			if err := m.Teams[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("teams" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("teams" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *RepositoryDetails) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *RepositoryDetails) UnmarshalBinary(b []byte) error {
	var res RepositoryDetails
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// RepositoryDetailsCollaboratorsItems0 repository details collaborators items0
//
// swagger:model RepositoryDetailsCollaboratorsItems0
type RepositoryDetailsCollaboratorsItems0 struct {

	// access
	// Min Length: 1
	Access string `json:"access,omitempty"`

	// name
	// Min Length: 1
	Name string `json:"name,omitempty"`
}

// Validate validates this repository details collaborators items0
func (m *RepositoryDetailsCollaboratorsItems0) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAccess(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *RepositoryDetailsCollaboratorsItems0) validateAccess(formats strfmt.Registry) error {
	if swag.IsZero(m.Access) { // not required
		return nil
	}

	if err := validate.MinLength("access", "body", m.Access, 1); err != nil {
		return err
	}

	return nil
}

func (m *RepositoryDetailsCollaboratorsItems0) validateName(formats strfmt.Registry) error {
	if swag.IsZero(m.Name) { // not required
		return nil
	}

	if err := validate.MinLength("name", "body", m.Name, 1); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this repository details collaborators items0 based on context it is used
func (m *RepositoryDetailsCollaboratorsItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *RepositoryDetailsCollaboratorsItems0) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *RepositoryDetailsCollaboratorsItems0) UnmarshalBinary(b []byte) error {
	var res RepositoryDetailsCollaboratorsItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// RepositoryDetailsTeamsItems0 repository details teams items0
//
// swagger:model RepositoryDetailsTeamsItems0
type RepositoryDetailsTeamsItems0 struct {

	// access
	// Min Length: 1
	Access string `json:"access,omitempty"`

	// name
	// Min Length: 1
	Name string `json:"name,omitempty"`
}

// Validate validates this repository details teams items0
func (m *RepositoryDetailsTeamsItems0) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAccess(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *RepositoryDetailsTeamsItems0) validateAccess(formats strfmt.Registry) error {
	if swag.IsZero(m.Access) { // not required
		return nil
	}

	if err := validate.MinLength("access", "body", m.Access, 1); err != nil {
		return err
	}

	return nil
}

func (m *RepositoryDetailsTeamsItems0) validateName(formats strfmt.Registry) error {
	if swag.IsZero(m.Name) { // not required
		return nil
	}

	if err := validate.MinLength("name", "body", m.Name, 1); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this repository details teams items0 based on context it is used
func (m *RepositoryDetailsTeamsItems0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *RepositoryDetailsTeamsItems0) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *RepositoryDetailsTeamsItems0) UnmarshalBinary(b []byte) error {
	var res RepositoryDetailsTeamsItems0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
