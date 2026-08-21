package adapter

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/samber/mo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"todoe/infra/audit/domain"
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

func (r *MongoRepository) collection() (*mongo.Collection, error) {
	either := r.getClient()
	if either.IsLeft() {
		return nil, either.MustLeft()
	}
	return either.MustRight().Database("todoe").Collection("audit_log"), nil
}

func (r *MongoRepository) Save(ctx context.Context, entry domain.AuditEntry) mo.Result[struct{}] {
	col, err := r.collection()
	if err != nil {
		return mo.Err[struct{}](err)
	}
	if _, err := col.InsertOne(ctx, entry); err != nil {
		return mo.Err[struct{}](err)
	}
	return mo.Ok(struct{}{})
}
