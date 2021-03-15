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

// PatchMachineDeploymentReader is a Reader for the PatchMachineDeployment structure.
type PatchMachineDeploymentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchMachineDeploymentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPatchMachineDeploymentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPatchMachineDeploymentUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewPatchMachineDeploymentForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewPatchMachineDeploymentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPatchMachineDeploymentOK creates a PatchMachineDeploymentOK with default headers values
func NewPatchMachineDeploymentOK() *PatchMachineDeploymentOK {
	return &PatchMachineDeploymentOK{}
}

/*PatchMachineDeploymentOK handles this case with default header values.

NodeDeployment
*/
type PatchMachineDeploymentOK struct {
	Payload *models.NodeDeployment
}

func (o *PatchMachineDeploymentOK) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] patchMachineDeploymentOK  %+v", 200, o.Payload)
}

func (o *PatchMachineDeploymentOK) GetPayload() *models.NodeDeployment {
	return o.Payload
}

func (o *PatchMachineDeploymentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.NodeDeployment)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchMachineDeploymentUnauthorized creates a PatchMachineDeploymentUnauthorized with default headers values
func NewPatchMachineDeploymentUnauthorized() *PatchMachineDeploymentUnauthorized {
	return &PatchMachineDeploymentUnauthorized{}
}

/*PatchMachineDeploymentUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type PatchMachineDeploymentUnauthorized struct {
}

func (o *PatchMachineDeploymentUnauthorized) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] patchMachineDeploymentUnauthorized ", 401)
}

func (o *PatchMachineDeploymentUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchMachineDeploymentForbidden creates a PatchMachineDeploymentForbidden with default headers values
func NewPatchMachineDeploymentForbidden() *PatchMachineDeploymentForbidden {
	return &PatchMachineDeploymentForbidden{}
}

/*PatchMachineDeploymentForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type PatchMachineDeploymentForbidden struct {
}

func (o *PatchMachineDeploymentForbidden) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] patchMachineDeploymentForbidden ", 403)
}

func (o *PatchMachineDeploymentForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPatchMachineDeploymentDefault creates a PatchMachineDeploymentDefault with default headers values
func NewPatchMachineDeploymentDefault(code int) *PatchMachineDeploymentDefault {
	return &PatchMachineDeploymentDefault{
		_statusCode: code,
	}
}

/*PatchMachineDeploymentDefault handles this case with default header values.

errorResponse
*/
type PatchMachineDeploymentDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the patch machine deployment default response
func (o *PatchMachineDeploymentDefault) Code() int {
	return o._statusCode
}

func (o *PatchMachineDeploymentDefault) Error() string {
	return fmt.Sprintf("[PATCH /api/v2/projects/{project_id}/clusters/{cluster_id}/machinedeployments/{machinedeployment_id}][%d] patchMachineDeployment default  %+v", o._statusCode, o.Payload)
}

func (o *PatchMachineDeploymentDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchMachineDeploymentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
