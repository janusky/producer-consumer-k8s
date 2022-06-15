package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	connectTimeout  = 40 * time.Second
	maxConnIdleTime = 3 * time.Minute
	minPoolSize     = 20
	maxPoolSize     = 300
)

type DocumentDB struct {
	Client *mongo.Client
	Ctx    context.Context
}

func NewDocumentDB(ctx context.Context, mongoURI, username, password string) (*DocumentDB, error) {
	if ctx == nil {
		ctx = context.TODO()
	}

	client, err := mongo.Connect(ctx, optionsClient(mongoURI, username, password))
	if err != nil {
		return nil, err
	}
	// client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	// if err != nil {
	// 	return nil, err
	// }

	// "go.mongodb.org/mongo-driver/mongo/readpref"
	// if err := client.Ping(ctx, readpref.Primary()); err != nil {
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &DocumentDB{
		Client: client,
		Ctx:    ctx,
	}, nil
}

// insertOne is a user defined method, used to insert documents into collection
// returns result of InsertOne and error if any.
func (docDb *DocumentDB) InsertOne(ctx context.Context, dataBase, col string,
	doc interface{}) (*mongo.InsertOneResult, error) {
	if ctx == nil {
		ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	}
	// select database and collection ith Client.Database method
	// and Database.Collection method
	collection := docDb.Client.Database(dataBase).Collection(col)

	// InsertOne accept two argument of type Context
	// and of empty interface
	result, err := collection.InsertOne(ctx, doc)
	return result, err
}

// ctxIn context.Context
func (docDb *DocumentDB) FindOne(ctx context.Context, dataBase, col string,
	filter interface{}, result interface{}) (interface{}, error) {
	if ctx == nil {
		ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	}

	collection := docDb.Client.Database(dataBase).Collection(col)

	err := collection.FindOne(ctx, filter).Decode(result)

	if err == mongo.ErrNoDocuments {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		return nil, nil
	}
	return result, err
}

// Find All Documents from a Collection
func (docDb *DocumentDB) FindAll(ctx context.Context, dataBase, col string,
	filter interface{}, result interface{}) (interface{}, error) {
	// var result []bson.D
	if ctx == nil {
		ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	}

	collection := docDb.Client.Database(dataBase).Collection(col)

	// err := collection.Find(ctx, filter, options.Find()).All(&result)
	cursor, err := collection.Find(ctx, filter, options.Find())
	err = cursor.All(ctx, result)

	return result, err
}

// query method returns a cursor and error.
//
// query is user defined method used to query MongoDB,
// that accepts mongo.client,context, database name,
// collection name, a query and field.
//
// database name and collection name is of type
// string. query is of type interface.
// field is of type interface, which limts
// the field being returned.
func (docDb *DocumentDB) Query(ctx context.Context,
	dataBase, col string, query, field interface{}) (result *mongo.Cursor, err error) {

	if ctx == nil {
		ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	}

	// select database and collection.
	collection := docDb.Client.Database(dataBase).Collection(col)

	// collection has an method Find,
	// that returns a mongo.cursor
	// based on query and field.
	result, err = collection.Find(ctx, query,
		options.Find().SetProjection(field))
	return
}

// This is a user defined method that accepts
// mongo.Client and context.Context
// This method used to ping the mongoDB, return error if any.
func (docDb *DocumentDB) Ping() error {
	// mongo.Client has Ping to ping mongoDB, deadline of
	// the Ping method will be determined by cxt
	// Ping method return error if any occored, then
	// the error can be handled.
	// "go.mongodb.org/mongo-driver/mongo/readpref"
	// if err := client.Ping(ctx, readpref.Primary()); err != nil {
	if err := docDb.Client.Ping(docDb.Ctx, nil); err != nil {
		return err
	}
	// fmt.Println("connected successfully")
	return nil
}

// This is a user defined method to close resourses.
// This method closes mongoDB connection and cancel context.
func (docDb *DocumentDB) Close() error {
	return docDb.Client.Disconnect(docDb.Ctx)
}

func optionsClient(mongoURI, username, password string) *options.ClientOptions {
	var opts *options.ClientOptions

	opts = options.Client().ApplyURI(mongoURI).
		SetConnectTimeout(connectTimeout).
		SetMaxConnIdleTime(maxConnIdleTime).
		SetMinPoolSize(minPoolSize).
		SetMaxPoolSize(maxPoolSize)

	if username != "" && password != "" {
		opts.SetAuth(options.Credential{
			Username: username,
			Password: password,
		})
	}
	return opts
}
