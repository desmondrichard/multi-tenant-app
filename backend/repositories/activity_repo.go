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

type ActivityRepo struct{}

func (r *ActivityRepo) coll() *mongo.Collection {
	return config.Database.Collection("activities")
}

func (r *ActivityRepo) Create(ctx context.Context, a *models.Activity) error {
	a.CreatedAt = time.Now()
	res, err := r.coll().InsertOne(ctx, a)
	if err != nil {
		return err
	}
	a.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ActivityRepo) ListByTask(ctx context.Context, orgID, taskID primitive.ObjectID) ([]models.Activity, error) {
	filter := bson.M{
		"organizationId": orgID,
		"taskId":         taskID,
	}
	// Sort newest first for the activity feed
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var activities []models.Activity
	if err := cursor.All(ctx, &activities); err != nil {
		return nil, err
	}
	return activities, nil
}
