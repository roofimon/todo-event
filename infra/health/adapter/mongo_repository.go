package adapter

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

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

func (r *MongoRepository) Ping(ctx context.Context) mo.Result[struct{}] {
	either := r.getClient()
	if either.IsLeft() {
		return mo.Err[struct{}](either.MustLeft())
	}
	if err := either.MustRight().Ping(ctx, nil); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}

func (r *MongoRepository) Disconnect(ctx context.Context) mo.Result[struct{}] {
	if !r.initialized.Load() || r.cached.IsLeft() {
		return mo.Ok(struct{}{})
	}
	if err := r.cached.MustRight().Disconnect(ctx); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}
