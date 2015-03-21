package mongostore

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// ErrInvalidID error when Session ID is invalid
	ErrInvalidID = errors.New("Invalid Session ID")
)

// Session struct stored in MongoDB
type Session struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	Data     string
	Modified time.Time
}

// MongoStore struct contains options and variables to interact with session settings
type MongoStore struct {
	Codecs     []securecookie.Codec
	Options    *sessions.Options
	Token      TokenGetSeter
	collection string
	dbSession  *mgo.Session
}

// NewMongoStore returns a new MongoStore.
// Set ensureTTL to true let the database auto-remove expired object by maxAge.
// This is using *mgo.Session instead of *.mgo.Collection because if the database goes offline and then
// comes back onlne we will need the session to refresh the database connection.
func NewMongoStore(s *mgo.Session, collectionName string, maxAge int, ensureTTL bool, keyPairs ...[]byte) *MongoStore {

	store := &MongoStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			MaxAge: maxAge,
		},
		Token:      &CookieToken{},
		dbSession:  s,
		collection: collectionName,
	}

	dbSess := s.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(store.collection)

	if ensureTTL {
		c.EnsureIndex(mgo.Index{
			Key:         []string{"modified"},
			Background:  true,
			Sparse:      true,
			ExpireAfter: time.Duration(maxAge) * time.Second,
		})
	}

	return store
}

// Get return a session for the given session name or creates a new one and return it.
func (m *MongoStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(m, name)
}

// New returns a session for the given session name without adding it to the registry.
func (m *MongoStore) New(r *http.Request, name string) (*sessions.Session, error) {

	session := sessions.NewSession(m, name)

	session.Options = &sessions.Options{
		Path:   m.Options.Path,
		MaxAge: m.Options.MaxAge,
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

// Save saves all sessions registered for the current request.
func (m *MongoStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {

	if session.Options.MaxAge < 0 {

		if err := m.delete(session); err != nil {
			return err
		}

		m.Token.SetToken(w, session.Name(), "", session.Options)
		return nil
	}

	if session.ID == "" {
		session.ID = bson.NewObjectId().Hex()
	}

	if err := m.upsert(session); err != nil {
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

func (m *MongoStore) load(session *sessions.Session) error {

	if !bson.IsObjectIdHex(session.ID) {
		return ErrInvalidID
	}

	dbSess := m.dbSession.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(m.collection)

	s := Session{}
	err := c.FindId(bson.ObjectIdHex(session.ID)).One(&s)
	if err != nil {
		return err
	}

	if err := securecookie.DecodeMulti(session.Name(), s.Data, &session.Values,
		m.Codecs...); err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) upsert(session *sessions.Session) error {

	if !bson.IsObjectIdHex(session.ID) {
		return ErrInvalidID
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

	s := Session{
		ID:       bson.ObjectIdHex(session.ID),
		Data:     encoded,
		Modified: modified,
	}

	dbSess := m.dbSession.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(m.collection)

	_, err = c.UpsertId(s.ID, &s)
	if err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) delete(session *sessions.Session) error {

	if !bson.IsObjectIdHex(session.ID) {
		return ErrInvalidID
	}

	dbSess := m.dbSession.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(m.collection)

	return c.RemoveId(bson.ObjectIdHex(session.ID))
}
