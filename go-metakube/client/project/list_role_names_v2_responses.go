// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

// ListRoleNamesV2Reader is a Reader for the ListRoleNamesV2 structure.
type ListRoleNamesV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListRoleNamesV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListRoleNamesV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListRoleNamesV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListRoleNamesV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListRoleNamesV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListRoleNamesV2OK creates a ListRoleNamesV2OK with default headers values
func NewListRoleNamesV2OK() *ListRoleNamesV2OK {
	return &ListRoleNamesV2OK{}
}

/*ListRoleNamesV2OK handles this case with default header values.

RoleName
*/
type ListRoleNamesV2OK struct {
	Payload []*models.RoleName
}

func (o *ListRoleNamesV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rolenames][%d] listRoleNamesV2OK  %+v", 200, o.Payload)
}

func (o *ListRoleNamesV2OK) GetPayload() []*models.RoleName {
	return o.Payload
}

func (o *ListRoleNamesV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListRoleNamesV2Unauthorized creates a ListRoleNamesV2Unauthorized with default headers values
func NewListRoleNamesV2Unauthorized() *ListRoleNamesV2Unauthorized {
	return &ListRoleNamesV2Unauthorized{}
}

/*ListRoleNamesV2Unauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListRoleNamesV2Unauthorized struct {
}

func (o *ListRoleNamesV2Unauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rolenames][%d] listRoleNamesV2Unauthorized ", 401)
}

func (o *ListRoleNamesV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListRoleNamesV2Forbidden creates a ListRoleNamesV2Forbidden with default headers values
func NewListRoleNamesV2Forbidden() *ListRoleNamesV2Forbidden {
	return &ListRoleNamesV2Forbidden{}
}

/*ListRoleNamesV2Forbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListRoleNamesV2Forbidden struct {
}

func (o *ListRoleNamesV2Forbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rolenames][%d] listRoleNamesV2Forbidden ", 403)
}

func (o *ListRoleNamesV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListRoleNamesV2Default creates a ListRoleNamesV2Default with default headers values
func NewListRoleNamesV2Default(code int) *ListRoleNamesV2Default {
	return &ListRoleNamesV2Default{
		_statusCode: code,
	}
}

/*ListRoleNamesV2Default handles this case with default header values.

errorResponse
*/
type ListRoleNamesV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list role names v2 default response
func (o *ListRoleNamesV2Default) Code() int {
	return o._statusCode
}

func (o *ListRoleNamesV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/rolenames][%d] listRoleNamesV2 default  %+v", o._statusCode, o.Payload)
}

func (o *ListRoleNamesV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListRoleNamesV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
