package resource_maintenance_cronjob

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// UpgradeState converts the version 0 singleton blocks into version 1 objects.
func (r *metakubeMaintenanceCronJob) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeMaintenanceCronJobState},
	}
}

func upgradeMaintenanceCronJobState(_ context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	if req.RawState == nil || len(req.RawState.JSON) == 0 {
		resp.Diagnostics.AddError("Missing maintenance cron job state", "The legacy state must contain JSON for the state upgrade.")
		return
	}

	var state map[string]any
	if err := json.Unmarshal(req.RawState.JSON, &state); err != nil {
		resp.Diagnostics.AddError("Invalid maintenance cron job state", fmt.Sprintf("Could not decode legacy state: %s", err))
		return
	}

	// Only these three levels were singleton blocks. In particular, keep the
	// inner options map and the timeouts block in their existing representations.
	parent := state
	var attributePath []string
	for _, key := range []string{"spec", "maintenance_job_template", "options"} {
		attributePath = append(attributePath, key)
		value := parent[key]
		if list, ok := value.([]any); ok {
			switch len(list) {
			case 0:
				value = nil
			case 1:
				value = list[0]
			default:
				resp.Diagnostics.AddError("Invalid maintenance cron job state", fmt.Sprintf("%s contains multiple blocks; expected at most one.", strings.Join(attributePath, ".")))
				return
			}
			parent[key] = value
		}
		if value == nil {
			break
		}
		object, ok := value.(map[string]any)
		if !ok {
			resp.Diagnostics.AddError("Invalid maintenance cron job state", fmt.Sprintf("%s must contain an object.", strings.Join(attributePath, ".")))
			return
		}
		parent = object
	}

	upgradedJSON, err := json.Marshal(state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upgrade maintenance cron job state", fmt.Sprintf("Could not encode upgraded state: %s", err))
		return
	}
	resp.DynamicValue = &tfprotov6.DynamicValue{JSON: upgradedJSON}
}
