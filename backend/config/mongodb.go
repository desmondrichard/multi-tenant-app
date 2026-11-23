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

	// Create unique index on organization slug
	_, err = Database.Collection("organizations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Println("⚠️ Failed to create unique index on slug:", err)
	} else {
		log.Println("✅ Unique index on organization slug verified")
	}

	// One PENDING invitation per email per organization (race-proof).
	// Accepted invitations are ignored by the index, so a removed person can be invited again later.
	_, err = Database.Collection("invitations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "organizationId", Value: 1}, {Key: "email", Value: 1}},
		Options: options.Index().
			SetUnique(true).
			SetName("uniq_pending_invite_per_org_email").
			SetPartialFilterExpression(bson.M{"status": "pending"}),
	})
	if err != nil {
		log.Println("⚠️ Could not create the pending-invite unique index (old duplicate pending invitations exist?). "+
			"The app still blocks duplicates in code. Cancel the old duplicate pending invitations, then restart:", err)
	} else {
		log.Println("✅ Unique index on pending invitations verified")
	}
}
