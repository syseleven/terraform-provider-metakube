package metakube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/models"
)

func stringifyResponseError(resErr error) string {
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
	if err = json.Unmarshal(rawData, &v); err == nil && errorMessage(v.Payload) != "" {
		return errorMessage(v.Payload)
	}
	return resErr.Error()
}

func errorMessage(e *models.ErrorResponse) string {
	if e != nil && e.Error != nil && e.Error.Message != nil {
		if len(e.Error.Additional) > 0 {
			return fmt.Sprintf("%s %v", *e.Error.Message, e.Error.Additional)
		}
		return *e.Error.Message
	}
	return ""
}

func strToPtr(s string) *string {
	return &s
}

func int32ToPtr(v int32) *int32 {
	return &v
}

func intToInt32Ptr(v int) *int32 {
	vv := int32(v)
	return &vv
}

func importResourceWithProjectAndClusterID(identifierName string) schema.StateContextFunc {
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

func importResourceWithOptionalProject(identifierName string) schema.StateContextFunc {
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
