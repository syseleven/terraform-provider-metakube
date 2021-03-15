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

// ListRoleBindingV2Reader is a Reader for the ListRoleBindingV2 structure.
type ListRoleBindingV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListRoleBindingV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListRoleBindingV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListRoleBindingV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListRoleBindingV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListRoleBindingV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListRoleBindingV2OK creates a ListRoleBindingV2OK with default headers values
func NewListRoleBindingV2OK() *ListRoleBindingV2OK {
	return &ListRoleBindingV2OK{}
}

/*ListRoleBindingV2OK handles this case with default header values.

RoleBinding
*/
type ListRoleBindingV2OK struct {
	Payload []*models.RoleBinding
}

func (o *ListRoleBindingV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/bindings][%d] listRoleBindingV2OK  %+v", 200, o.Payload)
}

func (o *ListRoleBindingV2OK) GetPayload() []*models.RoleBinding {
	return o.Payload
}

func (o *ListRoleBindingV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListRoleBindingV2Unauthorized creates a ListRoleBindingV2Unauthorized with default headers values
func NewListRoleBindingV2Unauthorized() *ListRoleBindingV2Unauthorized {
	return &ListRoleBindingV2Unauthorized{}
}

/*ListRoleBindingV2Unauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListRoleBindingV2Unauthorized struct {
}

func (o *ListRoleBindingV2Unauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/bindings][%d] listRoleBindingV2Unauthorized ", 401)
}

func (o *ListRoleBindingV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListRoleBindingV2Forbidden creates a ListRoleBindingV2Forbidden with default headers values
func NewListRoleBindingV2Forbidden() *ListRoleBindingV2Forbidden {
	return &ListRoleBindingV2Forbidden{}
}

/*ListRoleBindingV2Forbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListRoleBindingV2Forbidden struct {
}

func (o *ListRoleBindingV2Forbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/bindings][%d] listRoleBindingV2Forbidden ", 403)
}

func (o *ListRoleBindingV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListRoleBindingV2Default creates a ListRoleBindingV2Default with default headers values
func NewListRoleBindingV2Default(code int) *ListRoleBindingV2Default {
	return &ListRoleBindingV2Default{
		_statusCode: code,
	}
}

/*ListRoleBindingV2Default handles this case with default header values.

errorResponse
*/
type ListRoleBindingV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list role binding v2 default response
func (o *ListRoleBindingV2Default) Code() int {
	return o._statusCode
}

func (o *ListRoleBindingV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/bindings][%d] listRoleBindingV2 default  %+v", o._statusCode, o.Payload)
}

func (o *ListRoleBindingV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListRoleBindingV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
