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

// ListByOrg returns the workspace-wide activity feed (newest first) with optional
// filters. A zero userID means "all users"; action "" or "all" means "all actions".
func (r *ActivityRepo) ListByOrg(ctx context.Context, orgID primitive.ObjectID, action string, userID primitive.ObjectID, page, limit int64) ([]models.Activity, int64, error) {
	filter := bson.M{"organizationId": orgID}
	if action != "" && action != "all" {
		filter["action"] = action
	}
	if !userID.IsZero() {
		filter["userId"] = userID
	}

	total, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip((page - 1) * limit).
		SetLimit(limit)

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var activities []models.Activity
	if err := cursor.All(ctx, &activities); err != nil {
		return nil, 0, err
	}
	return activities, total, nil
}

// ListByTaskIDs returns the newest activities of ANY of the given tasks (used for project summaries)
func (r *ActivityRepo) ListByTaskIDs(ctx context.Context, orgID primitive.ObjectID, taskIDs []primitive.ObjectID, limit int64) ([]models.Activity, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	filter := bson.M{"organizationId": orgID, "taskId": bson.M{"$in": taskIDs}}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
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

// ListSince returns the workspace activities created after `since` (newest first)
func (r *ActivityRepo) ListSince(ctx context.Context, orgID primitive.ObjectID, since time.Time, limit int64) ([]models.Activity, error) {
	filter := bson.M{"organizationId": orgID, "createdAt": bson.M{"$gte": since}}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
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
