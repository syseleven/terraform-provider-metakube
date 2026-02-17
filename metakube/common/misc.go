package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

type RetryError struct {
	Err       error
	Retryable bool
}

func (e *RetryError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

func RetryableError(err error) *RetryError {
	return &RetryError{Err: err, Retryable: true}
}

func NonRetryableError(err error) *RetryError {
	return &RetryError{Err: err, Retryable: false}
}

type RetryFunc func() *RetryError

// RetryContext retries the given function until it succeeds, returns a non-retryable error,
// or the context deadline/timeout is exceeded.
// The polling interval starts at 500ms and uses exponential backoff with a max of 10s.
func RetryContext(ctx context.Context, timeout time.Duration, f RetryFunc) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.InitialInterval = 500 * time.Millisecond
	backoffPolicy.MaxInterval = 10 * time.Second

	var lastErr error
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		retryErr := f()
		if retryErr == nil {
			return struct{}{}, nil
		}

		lastErr = retryErr.Err
		if !retryErr.Retryable {
			return struct{}{}, backoff.Permanent(retryErr.Err)
		}

		return struct{}{}, retryErr.Err
	}, backoff.WithBackOff(backoffPolicy))
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		if lastErr != nil {
			return fmt.Errorf("timeout while waiting: %w", lastErr)
		}
		return err
	}
	if errors.Is(err, context.Canceled) {
		if lastErr != nil {
			return fmt.Errorf("context canceled: %w", lastErr)
		}
		return err
	}

	return err
}

const (
	// wait this time before starting resource checks
	RequestDelay = time.Second
)

func StringifyResponseError(resErr error) string {
	if resErr == nil {
		return ""
	}

	rawData, err := json.Marshal(resErr)
	if err != nil {
		return resErr.Error()
	}
	v := &struct {
		Payload *models.ErrorResponse
	}{}
	if err = json.Unmarshal(rawData, &v); err == nil && ErrorMessage(v.Payload) != "" {
		return ErrorMessage(v.Payload)
	}
	return resErr.Error()
}

func ErrorMessage(e *models.ErrorResponse) string {
	if e != nil && e.Error != nil && e.Error.Message != nil {
		if len(e.Error.Additional) > 0 {
			return fmt.Sprintf("%s %v", *e.Error.Message, e.Error.Additional)
		}
		return *e.Error.Message
	}
	return ""
}

func StrToPtr(s string) *string {
	return &s
}

func Int32ToPtr(v int32) *int32 {
	return &v
}

func IntToInt32Ptr(v int) *int32 {
	vv := int32(v)
	return &vv
}

func ImportResourceWithProjectAndClusterID(identifierName string) schema.StateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
		parts := strings.Split(d.Id(), ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("please provide resource identifier in format 'project_id:cluster_id:%s'", identifierName)
		}
		d.Set("project_id", parts[0])
		d.Set("cluster_id", parts[1])
		d.SetId(parts[2])
		return []*schema.ResourceData{d}, nil
	}
}

func ImportResourceWithOptionalProject(identifierName string) schema.StateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
		parts := strings.Split(d.Id(), ":")
		switch len(parts) {
		case 1:
			d.SetId(parts[0])
			return []*schema.ResourceData{d}, nil
		case 2:
			d.Set("project_id", parts[0])
			d.SetId(parts[1])
			return []*schema.ResourceData{d}, nil
		default:
			return nil, fmt.Errorf("please provide resource identifier in format 'project_id:%s' or '%s'", identifierName, identifierName)
		}
	}
}

func MetakubeResourceSystemLabelOrTag(key string) bool {
	for _, s := range []string{"labels.%", "metakube", "system-", "system/", "kubernetes.io"} {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

func MetakubeGetCluster(ctx context.Context, proj, cls string, k *MetaKubeProviderMeta) (*models.Cluster, bool, error) {
	p := project.NewGetClusterV2Params().
		WithContext(ctx).
		WithProjectID(proj).
		WithClusterID(cls)
	r, err := k.Client.Project.GetClusterV2(p, k.Auth)
	if err != nil {
		if e, ok := err.(*project.GetClusterV2Default); ok && e.Code() == http.StatusNotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to get cluster %s in project %s - error: %v", cls, proj, err)
	}

	return r.Payload, true, nil
}

func MetakubeResourceClusterFindProjectID(ctx context.Context, id string, meta *MetaKubeProviderMeta) (string, error) {
	res, err := meta.Client.Project.ListProjects(project.NewListProjectsParams(), meta.Auth)
	if err != nil {
		return "", fmt.Errorf("list projects: %v", err)
	}

	for _, project := range res.Payload {
		ok, err := metakubeResourceClusterBelongsToProject(ctx, project.ID, id, meta)
		if ok {
			return project.ID, nil
		}
		if err != nil {
			return "", err
		}
	}

	meta.Log.Infof("owner project for cluster with id '%s' not found", id)
	return "", nil
}

func metakubeResourceClusterBelongsToProject(ctx context.Context, prj, id string, meta *MetaKubeProviderMeta) (bool, error) {
	prms := project.NewListClustersV2Params().WithContext(ctx).WithProjectID(prj)
	res, err := meta.Client.Project.ListClustersV2(prms, meta.Auth)
	if err != nil {
		meta.Log.Debugf("lookup owner project: list clusters: %v", err)
		return false, fmt.Errorf("list clusters: %s", StringifyResponseError(err))
	}
	for _, item := range res.Payload {
		if item.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func MetakubeResourceClusterWaitForReady(ctx context.Context, k *MetaKubeProviderMeta, timeout time.Duration, projectID, clusterID, configuredVersion string) error {
	return RetryContext(ctx, timeout, func() *RetryError {

		p := project.NewGetClusterV2Params()
		p.SetContext(ctx)
		p.SetProjectID(projectID)
		p.SetClusterID(clusterID)

		cluster, err := k.Client.Project.GetClusterV2(p, k.Auth)
		if err != nil {
			return RetryableError(fmt.Errorf("unable to get cluster '%s': %s", clusterID, StringifyResponseError(err)))
		}

		p1 := project.NewGetClusterHealthV2Params()
		p1.SetContext(ctx)
		p1.SetProjectID(projectID)
		p1.SetClusterID(clusterID)

		clusterHealth, err := k.Client.Project.GetClusterHealthV2(p1, k.Auth)
		if err != nil {
			return RetryableError(fmt.Errorf("unable to get cluster '%s' health: %s", clusterID, StringifyResponseError(err)))
		}

		const up models.HealthStatus = 1
		if clusterHealth.Payload.Apiserver == up &&
			clusterHealth.Payload.CloudProviderInfrastructure == up &&
			clusterHealth.Payload.Controller == up &&
			clusterHealth.Payload.Etcd == up &&
			clusterHealth.Payload.MachineController == up &&
			clusterHealth.Payload.Scheduler == up &&
			clusterHealth.Payload.UserClusterControllerManager == up {
			if configuredVersion == "" {
				return nil
			} else if cluster.Payload.Status.Version == models.Semver(configuredVersion) {
				return nil
			}
		}

		k.Log.Debugf("waiting for cluster '%s' to be ready, %+v", clusterID, clusterHealth.Payload)
		return RetryableError(fmt.Errorf("waiting for cluster '%s' to be ready", clusterID))
	})
}
