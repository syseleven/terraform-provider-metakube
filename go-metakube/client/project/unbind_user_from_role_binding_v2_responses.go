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

// UnbindUserFromRoleBindingV2Reader is a Reader for the UnbindUserFromRoleBindingV2 structure.
type UnbindUserFromRoleBindingV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UnbindUserFromRoleBindingV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUnbindUserFromRoleBindingV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewUnbindUserFromRoleBindingV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewUnbindUserFromRoleBindingV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewUnbindUserFromRoleBindingV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUnbindUserFromRoleBindingV2OK creates a UnbindUserFromRoleBindingV2OK with default headers values
func NewUnbindUserFromRoleBindingV2OK() *UnbindUserFromRoleBindingV2OK {
	return &UnbindUserFromRoleBindingV2OK{}
}

/*UnbindUserFromRoleBindingV2OK handles this case with default header values.

RoleBinding
*/
type UnbindUserFromRoleBindingV2OK struct {
	Payload *models.RoleBinding
}

func (o *UnbindUserFromRoleBindingV2OK) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings][%d] unbindUserFromRoleBindingV2OK  %+v", 200, o.Payload)
}

func (o *UnbindUserFromRoleBindingV2OK) GetPayload() *models.RoleBinding {
	return o.Payload
}

func (o *UnbindUserFromRoleBindingV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.RoleBinding)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUnbindUserFromRoleBindingV2Unauthorized creates a UnbindUserFromRoleBindingV2Unauthorized with default headers values
func NewUnbindUserFromRoleBindingV2Unauthorized() *UnbindUserFromRoleBindingV2Unauthorized {
	return &UnbindUserFromRoleBindingV2Unauthorized{}
}

/*UnbindUserFromRoleBindingV2Unauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type UnbindUserFromRoleBindingV2Unauthorized struct {
}

func (o *UnbindUserFromRoleBindingV2Unauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings][%d] unbindUserFromRoleBindingV2Unauthorized ", 401)
}

func (o *UnbindUserFromRoleBindingV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUnbindUserFromRoleBindingV2Forbidden creates a UnbindUserFromRoleBindingV2Forbidden with default headers values
func NewUnbindUserFromRoleBindingV2Forbidden() *UnbindUserFromRoleBindingV2Forbidden {
	return &UnbindUserFromRoleBindingV2Forbidden{}
}

/*UnbindUserFromRoleBindingV2Forbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type UnbindUserFromRoleBindingV2Forbidden struct {
}

func (o *UnbindUserFromRoleBindingV2Forbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings][%d] unbindUserFromRoleBindingV2Forbidden ", 403)
}

func (o *UnbindUserFromRoleBindingV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewUnbindUserFromRoleBindingV2Default creates a UnbindUserFromRoleBindingV2Default with default headers values
func NewUnbindUserFromRoleBindingV2Default(code int) *UnbindUserFromRoleBindingV2Default {
	return &UnbindUserFromRoleBindingV2Default{
		_statusCode: code,
	}
}

/*UnbindUserFromRoleBindingV2Default handles this case with default header values.

errorResponse
*/
type UnbindUserFromRoleBindingV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the unbind user from role binding v2 default response
func (o *UnbindUserFromRoleBindingV2Default) Code() int {
	return o._statusCode
}

func (o *UnbindUserFromRoleBindingV2Default) Error() string {
	return fmt.Sprintf("[DELETE /api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings][%d] unbindUserFromRoleBindingV2 default  %+v", o._statusCode, o.Payload)
}

func (o *UnbindUserFromRoleBindingV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *UnbindUserFromRoleBindingV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
