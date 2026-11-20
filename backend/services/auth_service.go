package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/yourname/multi-tenant-pm/config"
	"github.com/yourname/multi-tenant-pm/models"
	"github.com/yourname/multi-tenant-pm/repositories"
	"github.com/yourname/multi-tenant-pm/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthService struct {
	Users *repositories.UserRepo
}

type RegisterInput struct {
	OrgName  string `json:"orgName" validate:"required,min=2"`
	Slug     string `json:"slug" validate:"required,min=2"`
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (string, *models.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	existing, err := s.Users.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}

	// A person can belong to MANY organizations (e.g. a member of one) and
	// create their own. If the email already has an account, they must prove
	// it is theirs with the account's password, and they can own only ONE org.
	if existing != nil {
		if !utils.CheckPassword(existing.PasswordHash, in.Password) {
			return "", nil, errors.New("an account with this email already exists - enter that account's existing password to create your own organization with it")
		}
		for _, m := range existing.Memberships {
			if m.Role == models.RoleOwner {
				return "", nil, errors.New("you already own an organization - a user can own only one organization")
			}
		}
	}

	// 1) Create org
	org := models.Organization{
		Name:      in.OrgName,
		Slug:      strings.ToLower(in.Slug),
		CreatedAt: time.Now(),
	}
	res, err := config.Database.Collection("organizations").InsertOne(ctx, org)
	if err != nil {
		// Handle duplicate slug error
		if mongo.IsDuplicateKeyError(err) {
			return "", nil, errors.New("workspace slug is already taken")
		}
		return "", nil, err
	}
	org.ID = res.InsertedID.(primitive.ObjectID)

	ownerMembership := models.Membership{
		OrganizationID:   org.ID,
		OrganizationName: org.Name,
		Role:             models.RoleOwner,
	}

	// 2a) Existing user: add the new organization as an extra membership
	if existing != nil {
		added, err := s.Users.AddOwnerMembership(ctx, existing.ID, ownerMembership)
		if err != nil || !added {
			// roll back the organization we just created
			config.Database.Collection("organizations").DeleteOne(ctx, bson.M{"_id": org.ID})
			if err != nil {
				return "", nil, err
			}
			return "", nil, errors.New("you already own an organization - a user can own only one organization")
		}
		existing.Memberships = append(existing.Memberships, ownerMembership)

		token, err := utils.GenerateToken(existing.ID.Hex(), org.ID.Hex(), org.Name, models.RoleOwner)
		return token, existing, err
	}

	// 2b) New user: hash pw and create the owner
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return "", nil, err
	}

	user := &models.User{
		Name:         in.Name,
		Email:        email,
		PasswordHash: hash,
		Memberships:  []models.Membership{ownerMembership},
	}
	if err := s.Users.Create(ctx, user); err != nil {
		config.Database.Collection("organizations").DeleteOne(ctx, bson.M{"_id": org.ID})
		return "", nil, err
	}

	// 3) Generate Token for the new workspace
	token, err := utils.GenerateToken(user.ID.Hex(), org.ID.Hex(), org.Name, models.RoleOwner)
	return token, user, err
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (string, *models.User, error) {
	user, err := s.Users.FindByEmail(ctx, strings.ToLower(in.Email))
	if err != nil {
		return "", nil, err
	}
	if user == nil || !utils.CheckPassword(user.PasswordHash, in.Password) {
		return "", nil, errors.New("invalid credentials")
	}

	// Get the first membership to log them into
	var activeMembership models.Membership
	if len(user.Memberships) > 0 {
		activeMembership = user.Memberships[0]
	} else {
		return "", nil, errors.New("user has no workspaces")
	}

	// Generate Token with active workspace
	token, err := utils.GenerateToken(user.ID.Hex(), activeMembership.OrganizationID.Hex(), activeMembership.OrganizationName, activeMembership.Role)
	return token, user, err
}
