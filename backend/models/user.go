package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RoleOwner   = "owner"
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleMember  = "member"
	RoleViewer  = "viewer"
)

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	Name           string             `bson:"name" json:"name"`
	Email          string             `bson:"email" json:"email"`
	PasswordHash   string             `bson:"passwordHash" json:"-"`
	Role           string             `bson:"role" json:"role"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}
