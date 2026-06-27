package audit

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/sony/gobreaker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Sink writes audit events to an external storage.
type Sink interface {
	Write(ctx context.Context, evt Event) error
}

// CloseableSink is a sink with shutdown support.
type CloseableSink interface {
	Sink
	Close(ctx context.Context) error
}

// MongoSink persists audit logs in MongoDB.
type MongoSink struct {
	client     *mongo.Client
	collection *mongo.Collection
	retention  time.Duration
	cb         *gobreaker.CircuitBreaker
}

type QueryFilters struct {
	ActorUserID  string
	Action       string
	ResourceType string
	ResourceID   string
	TenantID     string
	Status       string
	RequestID    string
	Search       string
	SortBy       string
	SortDir      string
	From         *time.Time
	To           *time.Time
}

// Reader queries persisted audit logs.
type Reader interface {
	List(ctx context.Context, filters QueryFilters, page, limit int) ([]Event, int64, error)
	GetByID(ctx context.Context, id string) (*Event, error)
}

func NewMongoSink(ctx context.Context, uri, dbName, collectionName string, timeout time.Duration, retentionDays int) (*MongoSink, error) {
	ctxDial, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(ctxDial, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctxDial, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "mongo-audit-sink",
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})
	return &MongoSink{
		client:     client,
		collection: client.Database(dbName).Collection(collectionName),
		retention:  time.Duration(retentionDays) * 24 * time.Hour,
		cb:         cb,
	}, nil
}

func (s *MongoSink) Write(ctx context.Context, evt Event) error {
	_, err := s.cb.Execute(func() (interface{}, error) {
		doc := bson.M{
			"_id":           evt.EventID,
			"event_id":      evt.EventID,
			"occurred_at":   evt.OccurredAt,
			"actor_user_id": evt.ActorUserID,
			"actor_realm":   evt.ActorRealm,
			"action":        evt.Action,
			"resource_type": evt.ResourceType,
			"resource_id":   evt.ResourceID,
			"tenant_id":     evt.TenantID,
			"request_id":    evt.RequestID,
			"ip":            evt.IP,
			"user_agent":    evt.UserAgent,
			"status":        evt.Status,
			"error_code":    evt.ErrorCode,
			"before":        evt.Before,
			"after":         evt.After,
			"metadata":      evt.Metadata,
			"source":        evt.Source,
		}
		if s.retention > 0 {
			doc["expire_at"] = evt.OccurredAt.Add(s.retention)
		}
		_, err := s.collection.UpdateByID(ctx, evt.EventID, bson.M{"$setOnInsert": doc}, options.Update().SetUpsert(true))
		return nil, err
	})
	return err
}

func (s *MongoSink) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "occurred_at", Value: -1}}},
		{Keys: bson.D{{Key: "actor_user_id", Value: 1}, {Key: "occurred_at", Value: -1}}},
		{Keys: bson.D{{Key: "resource_type", Value: 1}, {Key: "resource_id", Value: 1}, {Key: "occurred_at", Value: -1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "occurred_at", Value: -1}}},
		{Keys: bson.D{{Key: "action", Value: 1}, {Key: "occurred_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "occurred_at", Value: -1}}},
		{Keys: bson.D{{Key: "request_id", Value: 1}}},
	}
	if s.retention > 0 {
		models = append(models, mongo.IndexModel{
			Keys:    bson.D{{Key: "expire_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		})
	}
	_, err := s.collection.Indexes().CreateMany(ctx, models)
	return err
}

func (s *MongoSink) List(ctx context.Context, filters QueryFilters, page, limit int) ([]Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := bson.M{}
	if filters.ActorUserID != "" {
		query["actor_user_id"] = filters.ActorUserID
	}
	if filters.Action != "" {
		query["action"] = filters.Action
	}
	if filters.ResourceType != "" {
		query["resource_type"] = filters.ResourceType
	}
	if filters.ResourceID != "" {
		query["resource_id"] = filters.ResourceID
	}
	if filters.TenantID != "" {
		query["tenant_id"] = filters.TenantID
	}
	if filters.Status != "" {
		query["status"] = filters.Status
	}
	if filters.RequestID != "" {
		query["request_id"] = filters.RequestID
	}
	if filters.Search != "" {
		safeSearch := regexp.QuoteMeta(filters.Search)
		query["$or"] = bson.A{
			bson.M{"action": bson.M{"$regex": safeSearch, "$options": "i"}},
			bson.M{"resource_type": bson.M{"$regex": safeSearch, "$options": "i"}},
			bson.M{"resource_id": bson.M{"$regex": safeSearch, "$options": "i"}},
			bson.M{"actor_user_id": bson.M{"$regex": safeSearch, "$options": "i"}},
			bson.M{"request_id": bson.M{"$regex": safeSearch, "$options": "i"}},
		}
	}
	if filters.From != nil || filters.To != nil {
		rng := bson.M{}
		if filters.From != nil {
			rng["$gte"] = *filters.From
		}
		if filters.To != nil {
			rng["$lte"] = *filters.To
		}
		query["occurred_at"] = rng
	}

	total, err := s.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	sortField := "occurred_at"
	if filters.SortBy != "" {
		sortField = filters.SortBy
	}
	sortDir := -1
	if filters.SortDir == "ASC" {
		sortDir = 1
	}
	skip := int64((page - 1) * limit)
	findOpts := options.Find().SetSort(bson.D{{Key: sortField, Value: sortDir}}).SetSkip(skip).SetLimit(int64(limit))
	cursor, err := s.collection.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	list := make([]Event, 0, limit)
	for cursor.Next(ctx) {
		var evt Event
		if err := cursor.Decode(&evt); err != nil {
			return nil, 0, err
		}
		list = append(list, evt)
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *MongoSink) GetByID(ctx context.Context, id string) (*Event, error) {
	var evt Event
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&evt)
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func (s *MongoSink) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *MongoSink) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("mongo client is not configured")
	}
	return s.client.Ping(ctx, nil)
}
