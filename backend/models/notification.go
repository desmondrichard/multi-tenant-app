package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	UserID         primitive.ObjectID `bson:"userId" json:"userId"` // The user receiving the notification
	Title          string             `bson:"title" json:"title"`
	Message        string             `bson:"message" json:"message"`
	IsRead         bool               `bson:"isRead" json:"isRead"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}