package repositories

import (
	"context"
	"time"

	"github.com/yourname/multi-tenant-pm/config"
	"github.com/yourname/multi-tenant-pm/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvitationRepo struct{}

func (r *InvitationRepo) coll() *mongo.Collection {
	return config.Database.Collection("invitations")
}

func (r *InvitationRepo) Create(ctx context.Context, inv *models.Invitation) error {
	inv.CreatedAt = time.Now()
	inv.Status = "pending"
	res, err := r.coll().InsertOne(ctx, inv)
	if err != nil {
		return err
	}
	inv.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *InvitationRepo) FindByToken(ctx context.Context, token string) (*models.Invitation, error) {
	var inv models.Invitation
	err := r.coll().FindOne(ctx, bson.M{"token": token, "status": "pending"}).Decode(&inv)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &inv, err
}

func (r *InvitationRepo) MarkAccepted(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": "accepted"}})
	return err
}

// FindPending returns the pending invitation for this email in THIS organization (if any).
// The same email may have invitations in other organizations: only this org is checked.
func (r *InvitationRepo) FindPending(ctx context.Context, orgID primitive.ObjectID, email string) (*models.Invitation, error) {
	var inv models.Invitation
	opts := options.FindOne().SetCollation(&options.Collation{Locale: "en", Strength: 2}) // ignore case
	err := r.coll().FindOne(ctx, bson.M{
		"organizationId": orgID,
		"email":          email,
		"status":         "pending",
	}, opts).Decode(&inv)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &inv, err
}

// Delete removes an invitation (used when the invitation email could not be sent,
// so the admin can simply try again).
func (r *InvitationRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll().DeleteOne(ctx, bson.M{"_id": id})
	return err
}
