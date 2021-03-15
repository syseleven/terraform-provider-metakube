// Code generated by go-swagger; DO NOT EDIT.

package alibaba

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

// NewListAlibabaZonesNoCredentialsV2Params creates a new ListAlibabaZonesNoCredentialsV2Params object
// with the default values initialized.
func NewListAlibabaZonesNoCredentialsV2Params() *ListAlibabaZonesNoCredentialsV2Params {
	var ()
	return &ListAlibabaZonesNoCredentialsV2Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewListAlibabaZonesNoCredentialsV2ParamsWithTimeout creates a new ListAlibabaZonesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewListAlibabaZonesNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListAlibabaZonesNoCredentialsV2Params {
	var ()
	return &ListAlibabaZonesNoCredentialsV2Params{

		timeout: timeout,
	}
}

// NewListAlibabaZonesNoCredentialsV2ParamsWithContext creates a new ListAlibabaZonesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a context for a request
func NewListAlibabaZonesNoCredentialsV2ParamsWithContext(ctx context.Context) *ListAlibabaZonesNoCredentialsV2Params {
	var ()
	return &ListAlibabaZonesNoCredentialsV2Params{

		Context: ctx,
	}
}

// NewListAlibabaZonesNoCredentialsV2ParamsWithHTTPClient creates a new ListAlibabaZonesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListAlibabaZonesNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListAlibabaZonesNoCredentialsV2Params {
	var ()
	return &ListAlibabaZonesNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/*ListAlibabaZonesNoCredentialsV2Params contains all the parameters to send to the API endpoint
for the list alibaba zones no credentials v2 operation typically these are written to a http.Request
*/
type ListAlibabaZonesNoCredentialsV2Params struct {

	/*Region*/
	Region *string
	/*ClusterID*/
	ClusterID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListAlibabaZonesNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) WithContext(ctx context.Context) *ListAlibabaZonesNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListAlibabaZonesNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRegion adds the region to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) WithRegion(region *string) *ListAlibabaZonesNoCredentialsV2Params {
	o.SetRegion(region)
	return o
}

// SetRegion adds the region to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) SetRegion(region *string) {
	o.Region = region
}

// WithClusterID adds the clusterID to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) WithClusterID(clusterID string) *ListAlibabaZonesNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) WithProjectID(projectID string) *ListAlibabaZonesNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list alibaba zones no credentials v2 params
func (o *ListAlibabaZonesNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListAlibabaZonesNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Region != nil {

		// header param Region
		if err := r.SetHeaderParam("Region", *o.Region); err != nil {
			return err
		}

	}

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
