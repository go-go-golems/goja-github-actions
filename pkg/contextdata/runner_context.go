package contextdata

type RunnerContext struct {
	Workspace string `json:"workspace"`
	ActionPath string `json:"action_path,omitempty"`
	EventPath string `json:"event_path,omitempty"`
}
