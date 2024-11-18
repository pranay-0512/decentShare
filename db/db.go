package db

import (
	"context"
	"log"
	"p2p/config"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func InitDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(config.AppConfig.MONGO_CONNECTION_URL))
	if err != nil {
		log.Fatal("error connecting to database", err)
	}
	_ = client.Ping(ctx, readpref.Primary())
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}
