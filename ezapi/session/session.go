package session

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/94peter/vulpes/db/mgo"
)

const (
	sessionCollectionName = "ezapi_sessions"
	expireSeconds         = 3600 // 1 hour
)

func init() {
	mgo.RegisterIndex(sessionsCollection)
}

var sessionsCollection = mgo.NewCollectDef(sessionCollectionName, func() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "modified", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(expireSeconds), // 1 hour
		},
	}
})

func NewSession(opts ...SessionOption) *session {
	s := &session{
		Index: sessionsCollection,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type session struct {
	Modified  time.Time
	mgo.Index `bson:"-"`
	Data      string
	Id        bson.ObjectID `bson:"_id,omitempty"`
}

func (*session) Validate() error {
	return nil
}
func (s *session) GetId() any {
	return s.Id
}

func (s *session) SetId(id any) {
	if id, ok := id.(bson.ObjectID); ok {
		s.Id = id
	}
}

type SessionOption func(*session)

func WithId(id bson.ObjectID) SessionOption {
	return func(s *session) {
		s.Id = id
	}
}

func WithData(data string) SessionOption {
	return func(s *session) {
		s.Data = data
	}
}

func WithModified(modified time.Time) SessionOption {
	return func(s *session) {
		s.Modified = modified
	}
}
