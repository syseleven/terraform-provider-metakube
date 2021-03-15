// Code generated by go-swagger; DO NOT EDIT.

package addon

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

// GetAddonV2Reader is a Reader for the GetAddonV2 structure.
type GetAddonV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetAddonV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetAddonV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetAddonV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetAddonV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetAddonV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetAddonV2OK creates a GetAddonV2OK with default headers values
func NewGetAddonV2OK() *GetAddonV2OK {
	return &GetAddonV2OK{}
}

/*GetAddonV2OK handles this case with default header values.

Addon
*/
type GetAddonV2OK struct {
	Payload *models.Addon
}

func (o *GetAddonV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}][%d] getAddonV2OK  %+v", 200, o.Payload)
}

func (o *GetAddonV2OK) GetPayload() *models.Addon {
	return o.Payload
}

func (o *GetAddonV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Addon)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetAddonV2Unauthorized creates a GetAddonV2Unauthorized with default headers values
func NewGetAddonV2Unauthorized() *GetAddonV2Unauthorized {
	return &GetAddonV2Unauthorized{}
}

/*GetAddonV2Unauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type GetAddonV2Unauthorized struct {
}

func (o *GetAddonV2Unauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}][%d] getAddonV2Unauthorized ", 401)
}

func (o *GetAddonV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAddonV2Forbidden creates a GetAddonV2Forbidden with default headers values
func NewGetAddonV2Forbidden() *GetAddonV2Forbidden {
	return &GetAddonV2Forbidden{}
}

/*GetAddonV2Forbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type GetAddonV2Forbidden struct {
}

func (o *GetAddonV2Forbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}][%d] getAddonV2Forbidden ", 403)
}

func (o *GetAddonV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetAddonV2Default creates a GetAddonV2Default with default headers values
func NewGetAddonV2Default(code int) *GetAddonV2Default {
	return &GetAddonV2Default{
		_statusCode: code,
	}
}

/*GetAddonV2Default handles this case with default header values.

errorResponse
*/
type GetAddonV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get addon v2 default response
func (o *GetAddonV2Default) Code() int {
	return o._statusCode
}

func (o *GetAddonV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}][%d] getAddonV2 default  %+v", o._statusCode, o.Payload)
}

func (o *GetAddonV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetAddonV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
