package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ActionCreatedTask   = "created_task"
	ActionChangedStatus = "changed_status"
	ActionAddedComment  = "added_comment"
	ActionAssignedTask  = "assigned_task"
)

type Activity struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	TaskID         primitive.ObjectID `bson:"taskId" json:"taskId"`
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	Action         string             `bson:"action" json:"action"`
	Details        string             `bson:"details,omitempty" json:"details,omitempty"` // e.g., "from todo to in_progress"
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}
