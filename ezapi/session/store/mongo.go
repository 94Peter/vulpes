package store

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/94peter/vulpes/db/mgo"
	ginSession "github.com/gin-contrib/sessions"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mysession "github.com/94peter/vulpes/ezapi/session"
)

func NewMongoStore(maxAge int, keyPairs ...[]byte) Store {
	store := &mongoStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Opts: &sessions.Options{
			Path:   "/",
			MaxAge: maxAge,
		},
		Token: &CookieToken{},
	}

	store.MaxAge(maxAge)

	return store
}

type mongoStore struct {
	Codecs []securecookie.Codec
	Opts   *sessions.Options
	Token  TokenGetSeter
}

func (m *mongoStore) MaxAge(age int) {
	m.Opts.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range m.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// Get should return a cached session.
func (m *mongoStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(m, name)
}

// New should create and return a new session.
//
// Note that New should never return a nil session, even in the case of
// an error if using the Registry infrastructure to cache the session.
func (m *mongoStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(m, name)
	session.Options = &sessions.Options{
		Path:     m.Opts.Path,
		MaxAge:   m.Opts.MaxAge,
		Domain:   m.Opts.Domain,
		Secure:   m.Opts.Secure,
		HttpOnly: m.Opts.HttpOnly,
	}
	session.IsNew = true
	var err error
	if cook, errToken := m.Token.GetToken(r, name); errToken == nil {
		err = securecookie.DecodeMulti(name, cook, &session.ID, m.Codecs...)
		if err == nil {
			err = m.load(session)
			if err == nil {
				session.IsNew = false
			} else {
				err = nil
			}
		}
	}
	return session, err
}

// Save should persist session to the underlying store implementation.
func (m *mongoStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	if session.Options.MaxAge < 0 {
		if err := m.delete(ctx, session); err != nil {
			return err
		}
		m.Token.SetToken(w, session.Name(), "", session.Options)
		return nil
	}

	if session.ID == "" {
		session.ID = bson.NewObjectID().Hex()
	}

	if err := m.upsert(ctx, session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID,
		m.Codecs...)
	if err != nil {
		return err
	}

	m.Token.SetToken(w, session.Name(), encoded, session.Options)
	return nil
}

func (m *mongoStore) load(session *sessions.Session) error {
	oid, err := bson.ObjectIDFromHex(session.ID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s := mysession.NewSession(mysession.WithId(oid))
	err = mgo.FindById(ctx, s)
	if err != nil {
		return err
	}
	if err := securecookie.DecodeMulti(session.Name(), s.Data, &session.Values,
		m.Codecs...); err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) upsert(ctx context.Context, session *sessions.Session) error {
	oid, err := bson.ObjectIDFromHex(session.ID)
	if err != nil {
		return err
	}

	var modified time.Time
	if val, ok := session.Values["modified"]; ok {
		modified, ok = val.(time.Time)
		if !ok {
			return errors.New("mongostore: invalid modified value")
		}
	} else {
		modified = time.Now()
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		m.Codecs...)
	if err != nil {
		return err
	}
	s := mysession.NewSession(
		mysession.WithId(oid),
		mysession.WithData(encoded),
		mysession.WithModified(modified),
	)

	_, err = mgo.ReplaceOne(ctx, s, bson.M{"_id": s.Id}, options.Replace().SetUpsert(true))
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) delete(ctx context.Context, session *sessions.Session) error {
	oid, err := bson.ObjectIDFromHex(session.ID)
	if err != nil {
		return err
	}
	obj := mysession.NewSession(mysession.WithId(oid))
	_, err = mgo.DeleteById(ctx, obj)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) Options(options ginSession.Options) {
	m.Opts = options.ToGorillaOptions()
}

func (m *mongoStore) EncodeToken(name, id string) (string, error) {
	return securecookie.EncodeMulti(name, id, m.Codecs...)
}
