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
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

type RetryError struct {
	Err       error
	Retryable bool
}

// clusterReadinessError separates the user-facing explanation from its error chain.
type clusterReadinessError struct {
	message string
	cause   error
}

func (e *clusterReadinessError) Error() string {
	return e.message
}

func (e *clusterReadinessError) Unwrap() error {
	return e.cause
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
	if err := ctx.Err(); err != nil {
		return err
	}

	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.InitialInterval = 500 * time.Millisecond
	backoffPolicy.MaxInterval = 10 * time.Second

	var lastErr error
	// The context owns the timeout; disable backoff's default 15-minute limit.
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
	}, backoff.WithBackOff(backoffPolicy), backoff.WithMaxElapsedTime(0))
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		if lastErr != nil {
			return fmt.Errorf("timeout while waiting: %w: %w", err, lastErr)
		}
		return err
	}
	if errors.Is(err, context.Canceled) {
		if lastErr != nil {
			return fmt.Errorf("context canceled: %w: %w", err, lastErr)
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
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastObservation string
	retry := func(message string) *RetryError {
		lastObservation = message
		return RetryableError(errors.New(message))
	}
	err := RetryContext(ctx, timeout, func() *RetryError {

		p := project.NewGetClusterV2Params()
		p.SetContext(ctx)
		p.SetProjectID(projectID)
		p.SetClusterID(clusterID)

		cluster, err := k.Client.Project.GetClusterV2(p, k.Auth)
		if err != nil {
			if ctx.Err() != nil {
				return RetryableError(ctx.Err())
			}
			return retry(fmt.Sprintf("Could not read the cluster status from MetaKube: %s", StringifyResponseError(err)))
		}

		p1 := project.NewGetClusterHealthV2Params()
		p1.SetContext(ctx)
		p1.SetProjectID(projectID)
		p1.SetClusterID(clusterID)

		clusterHealth, err := k.Client.Project.GetClusterHealthV2(p1, k.Auth)
		if err != nil {
			if ctx.Err() != nil {
				return RetryableError(ctx.Err())
			}
			return retry(fmt.Sprintf("Could not read the cluster health from MetaKube: %s", StringifyResponseError(err)))
		}
		if cluster.Payload == nil || clusterHealth.Payload == nil {
			return retry("MetaKube returned no cluster status or health information.")
		}

		var currentVersion models.Semver
		if cluster.Payload.Status != nil {
			currentVersion = cluster.Payload.Status.Version
		}
		issues := clusterHealthIssues(clusterHealth.Payload)
		if configuredVersion != "" && currentVersion != models.Semver(configuredVersion) {
			if currentVersion == "" {
				issues = append(issues, fmt.Sprintf("- MetaKube has not reported the running Kubernetes version. Expected version: %s.", configuredVersion))
			} else {
				issues = append(issues, fmt.Sprintf("- The cluster reports Kubernetes version %s. Expected version: %s.", currentVersion, configuredVersion))
			}
		}
		if len(issues) == 0 {
			return nil
		}

		k.Log.Debugf("waiting for cluster '%s' to be ready: health=%+v, current version=%q, target version=%q", clusterID, *clusterHealth.Payload, currentVersion, configuredVersion)
		return retry("Last reported status:\n" + strings.Join(issues, "\n"))
	})
	if err == nil {
		return nil
	}

	message := fmt.Sprintf("Could not confirm that cluster %q is ready.", clusterID)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		message = fmt.Sprintf("Timed out waiting for cluster %q to become ready.", clusterID)
	case errors.Is(err, context.Canceled):
		message = fmt.Sprintf("Stopped waiting for cluster %q because the operation was canceled.", clusterID)
	}
	if lastObservation != "" {
		message += "\n\n" + lastObservation
	}
	message += fmt.Sprintf("\n\nOpen this cluster in project %q in the MetaKube dashboard.\nCheck the Events tab for provisioning errors.", projectID)
	if !errors.Is(err, context.Canceled) {
		message += "\nIf the cluster does not become ready, contact your MetaKube administrator or SysEleven support." +
			"\nInclude the cluster ID, project ID, and this error message."
	}
	return &clusterReadinessError{message: message, cause: err}
}

// clusterHealthIssues translates API health codes and lists only components that are not ready.
func clusterHealthIssues(health *models.ClusterHealth) []string {
	var issues []string
	for _, component := range []struct {
		name   string
		status models.HealthStatus
	}{
		{"Cloud infrastructure", health.CloudProviderInfrastructure},
		{"Kubernetes API server", health.Apiserver},
		{"Kubernetes controller", health.Controller},
		{"etcd", health.Etcd},
		{"Machine controller", health.MachineController},
		{"Kubernetes scheduler", health.Scheduler},
		{"User cluster controller manager", health.UserClusterControllerManager},
	} {
		// MetaKube defines 0 as down, 1 as up, and 2 as provisioning.
		var status string
		switch component.status {
		case 0:
			status = "not ready"
		case 1:
			continue
		case 2:
			status = "provisioning"
		default:
			status = "unrecognized health status"
		}
		issues = append(issues, fmt.Sprintf("- %s: %s.", component.name, status))
	}
	return issues
}
