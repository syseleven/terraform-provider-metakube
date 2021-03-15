// Code generated by go-swagger; DO NOT EDIT.

package vsphere

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

// ListVSphereNetworksNoCredentialsV2Reader is a Reader for the ListVSphereNetworksNoCredentialsV2 structure.
type ListVSphereNetworksNoCredentialsV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListVSphereNetworksNoCredentialsV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListVSphereNetworksNoCredentialsV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewListVSphereNetworksNoCredentialsV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListVSphereNetworksNoCredentialsV2OK creates a ListVSphereNetworksNoCredentialsV2OK with default headers values
func NewListVSphereNetworksNoCredentialsV2OK() *ListVSphereNetworksNoCredentialsV2OK {
	return &ListVSphereNetworksNoCredentialsV2OK{}
}

/*ListVSphereNetworksNoCredentialsV2OK handles this case with default header values.

VSphereNetwork
*/
type ListVSphereNetworksNoCredentialsV2OK struct {
	Payload []*models.VSphereNetwork
}

func (o *ListVSphereNetworksNoCredentialsV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/networks][%d] listVSphereNetworksNoCredentialsV2OK  %+v", 200, o.Payload)
}

func (o *ListVSphereNetworksNoCredentialsV2OK) GetPayload() []*models.VSphereNetwork {
	return o.Payload
}

func (o *ListVSphereNetworksNoCredentialsV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListVSphereNetworksNoCredentialsV2Default creates a ListVSphereNetworksNoCredentialsV2Default with default headers values
func NewListVSphereNetworksNoCredentialsV2Default(code int) *ListVSphereNetworksNoCredentialsV2Default {
	return &ListVSphereNetworksNoCredentialsV2Default{
		_statusCode: code,
	}
}

/*ListVSphereNetworksNoCredentialsV2Default handles this case with default header values.

errorResponse
*/
type ListVSphereNetworksNoCredentialsV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list v sphere networks no credentials v2 default response
func (o *ListVSphereNetworksNoCredentialsV2Default) Code() int {
	return o._statusCode
}

func (o *ListVSphereNetworksNoCredentialsV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/providers/vsphere/networks][%d] listVSphereNetworksNoCredentialsV2 default  %+v", o._statusCode, o.Payload)
}

func (o *ListVSphereNetworksNoCredentialsV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListVSphereNetworksNoCredentialsV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
