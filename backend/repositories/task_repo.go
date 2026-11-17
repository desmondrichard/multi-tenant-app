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

type TaskRepo struct{}

func (r *TaskRepo) coll() *mongo.Collection {
	return config.Database.Collection("tasks")
}

func (r *TaskRepo) Create(ctx context.Context, t *models.Task) error {
	t.CreatedAt = time.Now()
	res, err := r.coll().InsertOne(ctx, t)
	if err != nil {
		return err
	}
	t.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *TaskRepo) ListByProject(ctx context.Context, orgID, projectID primitive.ObjectID, search string, status string, priority string, sortBy string, page int64, limit int64) ([]models.Task, int64, error) {
	filter := bson.M{
		"organizationId": orgID,
		"projectId":      projectID,
	}

	if search != "" {
		filter["title"] = bson.M{"$regex": search, "$options": "i"}
	}
	if status != "" && status != "all" {
		filter["status"] = status
	}
	if priority != "" && priority != "all" {
		filter["priority"] = priority
	}

	totalCount, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	sortOrder := -1
	sortField := "createdAt"
	if sortBy == "oldest" {
		sortOrder = 1
	} else if sortBy == "priority" {
		sortField = "priority"
		sortOrder = 1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
		SetSkip((page - 1) * limit).
		SetLimit(limit)

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, err
	}
	return tasks, totalCount, nil
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, orgID, taskID primitive.ObjectID, status string) error {
	filter := bson.M{
		"_id":            taskID,
		"organizationId": orgID,
	}
	update := bson.M{"$set": bson.M{"status": status}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

func (r *TaskRepo) FindByID(ctx context.Context, orgID, taskID primitive.ObjectID) (*models.Task, error) {
	var task models.Task
	filter := bson.M{
		"_id":            taskID,
		"organizationId": orgID,
	}
	err := r.coll().FindOne(ctx, filter).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepo) ListByAssignee(ctx context.Context, orgID, userID primitive.ObjectID) ([]models.Task, error) {
	filter := bson.M{
		"organizationId": orgID,
		"assigneeId":     userID,
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepo) AddSubtask(ctx context.Context, orgID, taskID primitive.ObjectID, subtask models.Subtask) error {
	filter := bson.M{"_id": taskID, "organizationId": orgID}

	// The magic fix: Use $push. MongoDB automatically handles creating the array
	// if it doesn't exist, BUT if it's explicitly "null", it fails.
	// To bypass this, we can use a pipeline update or just fix the existing data.
	// For simplicity, let's use standard $push, and then fix the DB.
	update := bson.M{
		"$push": bson.M{"subtasks": subtask},
	}

	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

func (r *TaskRepo) ToggleSubtask(ctx context.Context, orgID, taskID primitive.ObjectID, subtaskID string, completed bool) error {
	filter := bson.M{
		"_id":            taskID,
		"organizationId": orgID,
		"subtasks.id":    subtaskID, // Matches bson:"id" from models.Subtask
	}
	update := bson.M{"$set": bson.M{"subtasks.$.completed": completed}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}
