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

type Membership struct {
	OrganizationID primitive.ObjectID `bson:"organizationId" json:"organizationId"`
	OrganizationName string           `bson:"organizationName" json:"organizationName"`
	Role           string             `bson:"role" json:"role"`
}

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Memberships  []Membership       `bson:"memberships" json:"memberships"` // <-- New array
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}