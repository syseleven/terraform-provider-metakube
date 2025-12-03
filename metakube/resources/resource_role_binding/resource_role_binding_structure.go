package resource_role_binding

import (
	"github.com/syseleven/go-metakube/models"
)

func metakubeRoleBindingExpandSubjects(p interface{}) []models.RoleUser {
	if p == nil {
		return nil
	}
	pp, ok := p.([]interface{})
	if !ok || len(pp) == 0 {
		return nil
	}

	var result []models.RoleUser
	for _, s := range pp {
		m := s.(map[string]interface{})
		if m["kind"].(string) == "user" {
			result = append(result, models.RoleUser{
				UserEmail: m["name"].(string),
			})
		} else if m["kind"].(string) == "group" {
			result = append(result, models.RoleUser{
				Group: m["name"].(string),
			})
		}
	}
	return result
}
