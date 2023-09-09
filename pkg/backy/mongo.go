package backy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const mongoConfigKey = "global.mongo"

func (opts *ConfigOpts) InitMongo() {

	if !opts.koanf.Bool(getMongoConfigKey("enabled")) {
		return
	}
	var (
		err    error
		client *mongo.Client
	)

	// TODO: Get uri and creditials from config
	host := opts.koanf.String(getMongoConfigKey("host"))
	port := opts.koanf.Int64(getMongoConfigKey("port"))

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongo://%s:%d", host, port)))
	if opts.koanf.Bool(getMongoConfigKey("prod")) {
		mongoEnvFileSet := opts.koanf.Exists(getMongoConfigKey("env"))
		if mongoEnvFileSet {
			getMongoConfigFromEnv(opts)
		}
		auth := options.Credential{}
		auth.Password = opts.koanf.String("global.mongo.password")
		auth.Username = opts.koanf.String("global.mongo.username")
		client, err = mongo.Connect(ctx, options.Client().SetAuth(auth).ApplyURI("mongodb://localhost:27017"))

	}
	if err != nil {
		opts.Logger.Fatal().Err(err).Send()
	}
	if err != nil {
		opts.Logger.Fatal().Err(err).Send()
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		opts.Logger.Fatal().Err(err).Send()
	}
	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		opts.Logger.Fatal().Err(err).Send()
	}
	fmt.Println(databases)
	backyDB := client.Database("backy")
	backyDB.CreateCollection(context.Background(), "cmds")
	backyDB.CreateCollection(context.Background(), "cmd-lists")
	backyDB.CreateCollection(context.Background(), "logs")
	opts.DB = backyDB
}

func getMongoConfigFromEnv(opts *ConfigOpts) error {
	mongoEnvFile, err := os.Open(opts.koanf.String("global.mongo.env"))
	if err != nil {
		return err
	}
	mongoMap, mongoErr := godotenv.Parse(mongoEnvFile)
	if mongoErr != nil {
		return err
	}
	mongoPW, mongoPWFound := mongoMap["MONGO_PASSWORD"]
	if !mongoPWFound {
		return errors.New("MONGO_PASSWORD not set in " + mongoEnvFile.Name())
	}
	mongoUser, mongoUserFound := mongoMap["MONGO_USER"]
	if !mongoUserFound {
		return errors.New("MONGO_PASSWORD not set in " + mongoEnvFile.Name())
	}
	opts.koanf.Set(mongoConfigKey+".password", mongoPW)
	opts.koanf.Set(mongoConfigKey+".username", mongoUser)

	return nil
}

func getMongoConfigKey(key string) string {
	return fmt.Sprintf("global.mongo.%s", key)
}
