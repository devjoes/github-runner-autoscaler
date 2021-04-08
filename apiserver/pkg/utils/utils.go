package utils

func ContainsStr(arr []string, i string) bool {
	for _, x := range arr {
		if x == i {
			return true
		}
	}
	return false
}

type WorkflowInfo struct {
	ID     int64    `json:"id,omitempty"`
	Name   string   `json:"name,omitempty"`
	Labels []string `json:"labels,omitempty"` //TODO: Rename to RunsOn
}
