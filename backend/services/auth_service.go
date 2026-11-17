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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthService struct {
	Users *repositories.UserRepo
}

type RegisterInput struct {
	OrgName  string `json:"orgName"  validate:"required,min=2"`
	Slug     string `json:"slug" validate:"required,min=2"`
	Name     string `json:"name"     validate:"required"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (string, *models.User, error) {
	// 1) Email uniqueness
	existing, err := s.Users.FindByEmail(ctx, strings.ToLower(in.Email))
	if err != nil {
		return "", nil, err
	}
	if existing != nil {
		return "", nil, errors.New("email already registered")
	}

	// 2) Create organization
	org := models.Organization{
		Name:      in.OrgName,
		Slug:      strings.ToLower(in.Slug),
		CreatedAt: time.Now(),
	}
	res, err := config.Database.Collection("organizations").InsertOne(ctx, org)
	if err != nil {
		return "", nil, err
	}
	org.ID = res.InsertedID.(primitive.ObjectID)

	// 3) Hash password
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return "", nil, err
	}

	// 4) Create user (owner)
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

	// 5) Issue token
	token, err := utils.GenerateToken(user.ID.Hex(), org.ID.Hex(), user.Role)
	return token, user, err
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (string, *models.User, error) {
	user, err := s.Users.FindByEmail(ctx, strings.ToLower(in.Email))
	if err != nil {
		return "", nil, err
	}
	if user == nil || !utils.CheckPassword(user.PasswordHash, in.Password) {
		// Deliberately vague error — see note below.
		return "", nil, errors.New("invalid credentials")
	}
	token, err := utils.GenerateToken(user.ID.Hex(), user.OrganizationID.Hex(), user.Role)
	return token, user, err
}
