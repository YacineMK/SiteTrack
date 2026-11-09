package rbac

type Resource struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

type Policy struct {
	Resource string
	Action   string
}

var Resources = []Resource{
	{Name: "materials", Actions: []string{"read", "write"}},
	{Name: "projects", Actions: []string{"read", "write"}},
	{Name: "tasks", Actions: []string{"read", "write"}},
	{Name: "project_store", Actions: []string{"read", "write"}},
	{Name: "project_deliveries", Actions: []string{"read", "write"}},
	{Name: "company_store", Actions: []string{"read", "write"}},
	{Name: "company_store_deliveries", Actions: []string{"read", "write"}},
	{Name: "roles", Actions: []string{"read", "write"}},
	{Name: "users", Actions: []string{"read", "write"}},
}

func Valid(resource, action string) bool {
	for _, candidate := range Resources {
		if candidate.Name != resource {
			continue
		}
		for _, allowed := range candidate.Actions {
			if allowed == action {
				return true
			}
		}
	}
	return false
}

func Allows(policies []Policy, resource, action string) bool {
	for _, policy := range policies {
		if policy.Resource != resource {
			continue
		}
		if policy.Action == action || (action == "read" && policy.Action == "write") {
			return true
		}
	}
	return false
}
