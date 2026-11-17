package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Invitation struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	Email          string             `bson:"email" json:"email"`
	Role           string             `bson:"role" json:"role"`
	Token          string             `bson:"token" json:"token"`   // The unique link token
	Status         string             `bson:"status" json:"status"` // "pending", "accepted"
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}
