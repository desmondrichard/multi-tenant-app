package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/yourname/multi-tenant-pm/config"
	"github.com/yourname/multi-tenant-pm/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepo struct{}

func (r *UserRepo) coll() *mongo.Collection {
	return config.Database.Collection("users")
}

func (r *UserRepo) Create(ctx context.Context, u *models.User) error {
	u.CreatedAt = time.Now()
	res, err := r.coll().InsertOne(ctx, u)
	if err != nil {
		return err
	}
	u.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.coll().FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var u models.User
	err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// NEW: ListByOrg returns all users belonging to a specific organization
func (r *UserRepo) ListByOrg(ctx context.Context, orgID primitive.ObjectID) ([]models.User, error) {
	cursor, err := r.coll().Find(ctx, bson.M{"organizationId": orgID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateRole changes a user's role in the database
func (r *UserRepo) UpdateRole(ctx context.Context, orgID, userID primitive.ObjectID, newRole string) error {
	filter := bson.M{
		"_id":            userID,
		"organizationId": orgID, // Ensure they belong to this org
	}
	update := bson.M{"$set": bson.M{"role": newRole}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

// RemoveUser deletes a user from the organization
func (r *UserRepo) RemoveUser(ctx context.Context, orgID, userID primitive.ObjectID) error {
	filter := bson.M{
		"_id":            userID,
		"organizationId": orgID,
	}
	_, err := r.coll().DeleteOne(ctx, filter)
	return err
}
