package metakube

import (
	"strings"

	"github.com/syseleven/go-metakube/models"
)

func metakubeClusterRoleBindingFlattenSubjects(in []*models.Subject) []interface{} {
	if len(in) == 0 {
		return nil
	}

	var result []interface{}
	for _, subject := range in {
		result = append(result, map[string]interface{}{
			"kind": strings.ToLower(subject.Kind),
			"name": subject.Name,
		})
	}

	return result
}

func metakubeRoleBindingExpandSubjects(p interface{}) []models.ClusterRoleUser {
	if p == nil {
		return nil
	}
	pp, ok := p.([]interface{})
	if !ok || len(pp) == 0 {
		return nil
	}

	var result []models.ClusterRoleUser
	for _, s := range pp {
		m := s.(map[string]interface{})
		if m["kind"].(string) == "user" {
			result = append(result, models.ClusterRoleUser{
				UserEmail: m["name"].(string),
			})
		} else if m["kind"].(string) == "group" {
			result = append(result, models.ClusterRoleUser{
				Group: m["name"].(string),
			})
		}
	}
	return result
}
