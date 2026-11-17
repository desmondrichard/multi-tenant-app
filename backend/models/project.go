package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	Name           string             `bson:"name" json:"name"`
	Key            string             `bson:"key" json:"key"`
	Description    string             `bson:"description" json:"description"`
	CreatedBy      primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}
