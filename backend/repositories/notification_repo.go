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

type NotificationRepo struct{}

func (r *NotificationRepo) coll() *mongo.Collection {
	return config.Database.Collection("notifications")
}

func (r *NotificationRepo) Create(ctx context.Context, n *models.Notification) error {
	n.CreatedAt = time.Now()
	res, err := r.coll().InsertOne(ctx, n)
	if err != nil {
		return err
	}
	n.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// ListByUser gets the latest 20 notifications for a specific user
func (r *NotificationRepo) ListByUser(ctx context.Context, orgID, userID primitive.ObjectID) ([]models.Notification, error) {
	filter := bson.M{
		"organizationId": orgID,
		"userId":         userID,
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(20)

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}
	return notifications, nil
}

// MarkAsRead updates a notification to read status
func (r *NotificationRepo) MarkAsRead(ctx context.Context, orgID, userID, notifID primitive.ObjectID) error {
	filter := bson.M{
		"_id":            notifID,
		"organizationId": orgID,
		"userId":         userID,
	}
	update := bson.M{"$set": bson.M{"isRead": true}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}
