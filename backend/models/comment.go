package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Comment struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	TaskID         primitive.ObjectID `bson:"taskId" json:"taskId"`
	SubtaskID      string             `bson:"subtaskId,omitempty" json:"subtaskId,omitempty"` // empty = normal task comment
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	Content        string             `bson:"content" json:"content"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}
