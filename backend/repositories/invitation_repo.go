package repositories

import (
	"context"
	"time"

	"github.com/yourname/multi-tenant-pm/config"
	"github.com/yourname/multi-tenant-pm/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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