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
	return &u, err
}

func (r *UserRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var u models.User
	err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &u, err
}

// FIXED: Search inside the memberships array
func (r *UserRepo) ListByOrg(ctx context.Context, orgID primitive.ObjectID) ([]models.User, error) {
	cursor, err := r.coll().Find(ctx, bson.M{"memberships.organizationId": orgID})
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

// Added for saving the modified user (used in AcceptInvite)
func (r *UserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": bson.M{"memberships": user.Memberships}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

// Added for updating a role inside the memberships array
func (r *UserRepo) UpdateRoleInMemberships(ctx context.Context, orgID, userID primitive.ObjectID, newRole string) error {
	filter := bson.M{"_id": userID, "memberships.organizationId": orgID}
	update := bson.M{"$set": bson.M{"memberships.$.role": newRole}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

// Added for removing a user from a specific workspace
func (r *UserRepo) RemoveMembership(ctx context.Context, orgID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	update := bson.M{"$pull": bson.M{"memberships": bson.M{"organizationId": orgID}}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

// FindByIDs returns users by their IDs (used to show names in the activity feed)
func (r *UserRepo) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	cursor, err := r.coll().Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
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
