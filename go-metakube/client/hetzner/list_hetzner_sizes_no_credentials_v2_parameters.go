// Code generated by go-swagger; DO NOT EDIT.

package hetzner

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewListHetznerSizesNoCredentialsV2Params creates a new ListHetznerSizesNoCredentialsV2Params object
// with the default values initialized.
func NewListHetznerSizesNoCredentialsV2Params() *ListHetznerSizesNoCredentialsV2Params {
	var ()
	return &ListHetznerSizesNoCredentialsV2Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewListHetznerSizesNoCredentialsV2ParamsWithTimeout creates a new ListHetznerSizesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewListHetznerSizesNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListHetznerSizesNoCredentialsV2Params {
	var ()
	return &ListHetznerSizesNoCredentialsV2Params{

		timeout: timeout,
	}
}

// NewListHetznerSizesNoCredentialsV2ParamsWithContext creates a new ListHetznerSizesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a context for a request
func NewListHetznerSizesNoCredentialsV2ParamsWithContext(ctx context.Context) *ListHetznerSizesNoCredentialsV2Params {
	var ()
	return &ListHetznerSizesNoCredentialsV2Params{

		Context: ctx,
	}
}

// NewListHetznerSizesNoCredentialsV2ParamsWithHTTPClient creates a new ListHetznerSizesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListHetznerSizesNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListHetznerSizesNoCredentialsV2Params {
	var ()
	return &ListHetznerSizesNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/*ListHetznerSizesNoCredentialsV2Params contains all the parameters to send to the API endpoint
for the list hetzner sizes no credentials v2 operation typically these are written to a http.Request
*/
type ListHetznerSizesNoCredentialsV2Params struct {

	/*ClusterID*/
	ClusterID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListHetznerSizesNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) WithContext(ctx context.Context) *ListHetznerSizesNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListHetznerSizesNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) WithClusterID(clusterID string) *ListHetznerSizesNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) WithProjectID(projectID string) *ListHetznerSizesNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list hetzner sizes no credentials v2 params
func (o *ListHetznerSizesNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListHetznerSizesNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
