package config

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client   *mongo.Client
	Database *mongo.Database
)

func ConnectMongo() {
	uri := os.Getenv("MONGO_URI")
	dbName := os.Getenv("MONGO_DB")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("❌ Mongo connect error: ", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("❌ Mongo ping error: ", err)
	}

	Client = client
	Database = client.Database(dbName)
	log.Println("✅ MongoDB connected:", dbName)

	ensureIndexes(ctx)
}

// ensureIndexes enforces unique organization names/slugs at the database level.
// If old duplicate data already exists, index creation fails and we only log a
// warning (the check in AuthService.Register still blocks new duplicates).
func ensureIndexes(ctx context.Context) {
	orgs := Database.Collection("organizations")
	models := []mongo.IndexModel{
		{
			// Case-insensitive unique name
			Keys: bson.D{{Key: "name", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("uniq_org_name_ci").
				SetCollation(&options.Collation{Locale: "en", Strength: 2}),
		},
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_org_slug"),
		},
	}
	if _, err := orgs.Indexes().CreateMany(ctx, models); err != nil {
		log.Println("⚠️ Could not create unique organization indexes (existing duplicates?):", err)
	}
}
