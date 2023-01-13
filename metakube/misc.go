package metakube

import (
	"encoding/json"
	"fmt"

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

func int64ToPtr(v int) *int64 {
	vv := int64(v)
	return &vv
}
