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

type ProjectRepo struct{}

func (r *ProjectRepo) coll() *mongo.Collection {
	return config.Database.Collection("projects")
}

func (r *ProjectRepo) Create(ctx context.Context, p *models.Project) error {
	p.CreatedAt = time.Now()
	res, err := r.coll().InsertOne(ctx, p)
	if err != nil {
		return err
	}
	p.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ProjectRepo) List(ctx context.Context, orgID primitive.ObjectID) ([]models.Project, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := r.coll().Find(ctx, bson.M{"organizationId": orgID, "deletedAt": nil}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []models.Project
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// SoftDelete hides a project AND all of its tasks (sets deletedAt),
// but nothing is removed from the database.
// Returns mongo.ErrNoDocuments if the project doesn't exist or is already deleted.
func (r *ProjectRepo) SoftDelete(ctx context.Context, orgID, projectID, userID primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{
		"_id":            projectID,
		"organizationId": orgID,
		"deletedAt":      nil,
	}
	update := bson.M{"$set": bson.M{
		"deletedAt": now,
		"deletedBy": userID,
	}}
	res, err := r.coll().UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	// Cascade: hide every task that belongs to this project
	_, err = config.Database.Collection("tasks").UpdateMany(ctx,
		bson.M{"organizationId": orgID, "projectId": projectID, "deletedAt": nil},
		bson.M{"$set": bson.M{"deletedAt": now, "deletedBy": userID}},
	)
	return err
}
