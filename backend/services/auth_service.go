package services

import (
	"context"
	"errors"
	"regexp"
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
	existing, err := s.Users.FindByEmail(ctx, strings.ToLower(in.Email))
	if err != nil {
		return "", nil, err
	}
	if existing != nil {
		return "", nil, errors.New("email already registered")
	}

	// 1) Make sure the organization name and slug are not already taken
	orgName := strings.TrimSpace(in.OrgName)
	orgSlug := strings.ToLower(strings.TrimSpace(in.Slug))
	orgs := config.Database.Collection("organizations")

	// Name check is case-insensitive: "ABC Tech" and "abc tech" are the same
	nameFilter := bson.M{"name": bson.M{
		"$regex":   "^" + regexp.QuoteMeta(orgName) + "$",
		"$options": "i",
	}}
	if n, err := orgs.CountDocuments(ctx, nameFilter); err != nil {
		return "", nil, err
	} else if n > 0 {
		return "", nil, errors.New("organization name already exists, please choose a different name")
	}
	if n, err := orgs.CountDocuments(ctx, bson.M{"slug": orgSlug}); err != nil {
		return "", nil, err
	} else if n > 0 {
		return "", nil, errors.New("workspace slug already taken, please choose a different slug")
	}

	// 2) Create org
	org := models.Organization{
		Name:      orgName,
		Slug:      orgSlug,
		CreatedAt: time.Now(),
	}
	res, err := orgs.InsertOne(ctx, org)
	if err != nil {
		// Safety net for two people registering the same name at the same moment
		if mongo.IsDuplicateKeyError(err) {
			return "", nil, errors.New("organization name or slug already exists")
		}
		return "", nil, err
	}
	org.ID = res.InsertedID.(primitive.ObjectID)

	// 3) Hash pw
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return "", nil, err
	}

	// 4) Create owner user
	user := &models.User{
		OrganizationID: org.ID,
		Name:           in.Name,
		Email:          strings.ToLower(in.Email),
		PasswordHash:   hash,
		Role:           models.RoleOwner,
	}
	if err := s.Users.Create(ctx, user); err != nil {
		return "", nil, err
	}

	// 5) Generate Token with orgName
	token, err := utils.GenerateToken(user.ID.Hex(), org.ID.Hex(), org.Name, user.Role)
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

	// Fetch organization name for the token
	var org models.Organization
	config.Database.Collection("organizations").FindOne(ctx, bson.M{"_id": user.OrganizationID}).Decode(&org)

	// Generate Token with orgName
	token, err := utils.GenerateToken(user.ID.Hex(), user.OrganizationID.Hex(), org.Name, user.Role)
	return token, user, err
}
