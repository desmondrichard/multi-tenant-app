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

type CommentRepo struct{}

func (r *CommentRepo) coll() *mongo.Collection {
	return config.Database.Collection("comments")
}

func (r *CommentRepo) Create(ctx context.Context, c *models.Comment) error {
	c.CreatedAt = time.Now()
	res, err := r.coll().InsertOne(ctx, c)
	if err != nil {
		return err
	}
	c.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// ListByTask returns comments for a task. If subtaskID is empty, only
// task-level comments are returned (subtask comments are excluded).
// If subtaskID is set, only comments of that subtask are returned.
func (r *CommentRepo) ListByTask(ctx context.Context, orgID, taskID primitive.ObjectID, subtaskID string) ([]models.Comment, error) {
	filter := bson.M{
		"organizationId": orgID,
		"taskId":         taskID,
	}
	if subtaskID == "" {
		filter["subtaskId"] = bson.M{"$in": bson.A{nil, ""}} // nil matches missing field
	} else {
		filter["subtaskId"] = subtaskID
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}) // Oldest first

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comments []models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}
