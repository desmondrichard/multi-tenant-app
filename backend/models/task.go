package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	StatusTodo        = "todo"
	StatusInProgress  = "in_progress"
	StatusUnderReview = "under_review"
	StatusDone        = "done"
)

const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

type Subtask struct {
	ID          string `bson:"id" json:"id"` // Standard string ID for arrays
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description" json:"description"`
	Completed   bool   `bson:"completed" json:"completed"`
}

type Task struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	ProjectID      primitive.ObjectID `bson:"projectId" json:"projectId"`
	Title          string             `bson:"title" json:"title"`
	Description    string             `bson:"description" json:"description"`
	Status         string             `bson:"status" json:"status"`
	Priority       string             `bson:"priority" json:"priority"`
	AssigneeID     primitive.ObjectID `bson:"assigneeId,omitempty" json:"assigneeId"`
	CreatedBy      primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	DueDate        *time.Time         `bson:"dueDate,omitempty" json:"dueDate"`
	Subtasks       []Subtask          `bson:"subtasks" json:"subtasks"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`

	// Soft delete: the document stays in the database, it is just hidden
	DeletedAt *time.Time         `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	DeletedBy primitive.ObjectID `bson:"deletedBy,omitempty" json:"deletedBy,omitempty"`
}
