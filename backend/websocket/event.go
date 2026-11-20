package websocket

// Event is the message that travels through Redis pub/sub and then to
// the browsers over WebSocket.
//
// Targeting rules (checked by the Hub for every connected client):
//   - OrgID set        -> only clients connected to that workspace get it
//   - TargetUser set   -> only that user's connections get it
type Event struct {
	Type       string `json:"type"` // NEW_COMMENT, TASKS_UPDATED, PROJECTS_UPDATED, ROLE_CHANGED, FORCE_LOGOUT
	OrgID      string `json:"orgId,omitempty"`
	TargetUser string `json:"targetUserId,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	TaskID     string `json:"taskId,omitempty"`
	SubtaskID  string `json:"subtaskId,omitempty"`
}
