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
		"deletedAt":      nil, // hide soft-deleted tasks
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
		"deletedAt":      nil,
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
		"deletedAt":      nil,
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
		"deletedAt":      nil,
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
	filter := bson.M{"_id": taskID, "organizationId": orgID, "deletedAt": nil}

	// If subtasks is null or doesn't exist, $push will fail.
	// We use an aggregation pipeline to check and initialize the array first.
	update := bson.A{
		bson.M{
			"$set": bson.M{
				"subtasks": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$isArray": "$subtasks"},
						"then": "$subtasks",
						"else": bson.A{},
					},
				},
			},
		},
		bson.M{
			"$set": bson.M{
				"subtasks": bson.M{
					"$concatArrays": []interface{}{
						"$subtasks",
						bson.A{subtask},
					},
				},
			},
		},
	}

	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

func (r *TaskRepo) ToggleSubtask(ctx context.Context, orgID, taskID primitive.ObjectID, subtaskID string, completed bool) error {
	filter := bson.M{
		"_id":            taskID,
		"organizationId": orgID,
		"deletedAt":      nil,
		"subtasks.id":    subtaskID, // Matches bson:"id" from models.Subtask
	}
	update := bson.M{"$set": bson.M{"subtasks.$.completed": completed}}
	_, err := r.coll().UpdateOne(ctx, filter, update)
	return err
}

// UpdateSubtaskDetails updates the title and description of one subtask
func (r *TaskRepo) UpdateSubtaskDetails(ctx context.Context, orgID, taskID primitive.ObjectID, subtaskID, title, description string) error {
	filter := bson.M{
		"_id":            taskID,
		"organizationId": orgID,
		"deletedAt":      nil,
		"subtasks.id":    subtaskID,
	}
	update := bson.M{"$set": bson.M{
		"subtasks.$.title":       title,
		"subtasks.$.description": description,
	}}
	res, err := r.coll().UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// HasSubtask checks that a subtask belongs to the task
func (r *TaskRepo) HasSubtask(ctx context.Context, orgID, taskID primitive.ObjectID, subtaskID string) (*models.Subtask, error) {
	task, err := r.FindByID(ctx, orgID, taskID)
	if err != nil || task == nil {
		return nil, err
	}
	for i := range task.Subtasks {
		if task.Subtasks[i].ID == subtaskID {
			return &task.Subtasks[i], nil
		}
	}
	return nil, nil
}

// SoftDelete hides a task (sets deletedAt) but keeps it in the database.
// Returns mongo.ErrNoDocuments if the task doesn't exist or is already deleted.
func (r *TaskRepo) SoftDelete(ctx context.Context, orgID, taskID, userID primitive.ObjectID) error {
	filter := bson.M{
		"_id":            taskID,
		"organizationId": orgID,
		"deletedAt":      nil,
	}
	update := bson.M{"$set": bson.M{
		"deletedAt": time.Now(),
		"deletedBy": userID,
	}}
	res, err := r.coll().UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// FindByIDs returns tasks (INCLUDING soft-deleted ones) so old activity can still show a title
func (r *TaskRepo) FindByIDs(ctx context.Context, orgID primitive.ObjectID, ids []primitive.ObjectID) ([]models.Task, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	cursor, err := r.coll().Find(ctx, bson.M{
		"organizationId": orgID,
		"_id":            bson.M{"$in": ids},
	})
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

// ListByOrg returns the newest tasks of the whole workspace (not deleted) plus the total count.
// Used by the AI chat helper to know about every project's tasks.
func (r *TaskRepo) ListByOrg(ctx context.Context, orgID primitive.ObjectID, limit int64) ([]models.Task, int64, error) {
	filter := bson.M{"organizationId": orgID, "deletedAt": nil}

	total, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit)
	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// UnassignUser removes a user as assignee from ALL tasks of the workspace.
// Called when the user is removed from the workspace, so no task stays
// assigned to someone who can no longer see it.
// Returns how many tasks were changed.
func (r *TaskRepo) UnassignUser(ctx context.Context, orgID, userID primitive.ObjectID) (int64, error) {
	res, err := r.coll().UpdateMany(ctx,
		bson.M{"organizationId": orgID, "assigneeId": userID},
		bson.M{"$unset": bson.M{"assigneeId": ""}},
	)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}