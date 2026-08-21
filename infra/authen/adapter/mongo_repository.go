package adapter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"todoe/infra/authen/domain"
)

type StoredEvent struct {
	ID          bson.ObjectID `bson:"_id"`
	AggregateID bson.ObjectID `bson:"aggregate_id"`
	Type        string        `bson:"type"`
	Payload     bson.Raw      `bson:"payload"`
	CreatedAt   time.Time     `bson:"created_at"`
}

type MongoRepository struct {
	clientIO    mo.IOEither[*mongo.Client]
	once        sync.Once
	cached      mo.Either[error, *mongo.Client]
	initialized atomic.Bool
}

func NewMongoRepository(clientIO mo.IOEither[*mongo.Client]) *MongoRepository {
	return &MongoRepository{clientIO: clientIO}
}

func (r *MongoRepository) getClient() mo.Either[error, *mongo.Client] {
	r.once.Do(func() {
		r.cached = r.clientIO.Run()
		r.initialized.Store(true)
	})
	return r.cached
}

func (r *MongoRepository) db() (*mongo.Database, error) {
	either := r.getClient()
	if either.IsLeft() {
		return nil, either.MustLeft()
	}
	return either.MustRight().Database("todoe"), nil
}

func (r *MongoRepository) Append(ctx context.Context, aggregateID bson.ObjectID, eventType string, payload any) mo.Result[struct{}] {
	db, err := r.db()
	if err != nil {
		return mo.Err[struct{}](err)
	}
	raw, err := bson.Marshal(payload)
	if err != nil {
		return mo.Err[struct{}](err)
	}
	e := StoredEvent{
		ID:          bson.NewObjectID(),
		AggregateID: aggregateID,
		Type:        eventType,
		Payload:     raw,
		CreatedAt:   time.Now(),
	}
	if _, err := db.Collection("auth_events").InsertOne(ctx, e); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MongoRepository) CreateCredential(ctx context.Context, cred domain.Credential) mo.Result[struct{}] {
	db, err := r.db()
	if err != nil {
		return mo.Err[struct{}](err)
	}
	if _, err := db.Collection("auth_credentials").InsertOne(ctx, cred); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MongoRepository) FindCredentialByEmail(ctx context.Context, email string) mo.Result[domain.Credential] {
	db, err := r.db()
	if err != nil {
		return mo.Err[domain.Credential](err)
	}
	var cred domain.Credential
	if err := db.Collection("auth_credentials").FindOne(ctx, bson.D{{Key: "email", Value: email}}).Decode(&cred); err != nil {
		return mo.Err[domain.Credential](err)
	}
	return mo.Ok(cred)
}

func (r *MongoRepository) UpsertSession(ctx context.Context, session domain.Session) mo.Result[struct{}] {
	db, err := r.db()
	if err != nil {
		return mo.Err[struct{}](err)
	}
	filter := bson.D{{Key: "_id", Value: session.ID}}
	update := bson.D{{Key: "$set", Value: session}}
	if _, err := db.Collection("auth_sessions").UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MongoRepository) FindActiveSessionByToken(ctx context.Context, token string) mo.Result[domain.Session] {
	db, err := r.db()
	if err != nil {
		return mo.Err[domain.Session](err)
	}
	filter := bson.D{
		{Key: "token", Value: token},
		{Key: "active", Value: true},
	}
	var session domain.Session
	if err := db.Collection("auth_sessions").FindOne(ctx, filter).Decode(&session); err != nil {
		return mo.Err[domain.Session](err)
	}
	return mo.Ok(session)
}

func (r *MongoRepository) DeactivateSession(ctx context.Context, token string) mo.Result[struct{}] {
	db, err := r.db()
	if err != nil {
		return mo.Err[struct{}](err)
	}
	filter := bson.D{{Key: "token", Value: token}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "active", Value: false}}}}
	if _, err := db.Collection("auth_sessions").UpdateOne(ctx, filter, update); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}
