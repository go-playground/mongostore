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

const (
	lastAccessedField = "lastAccessed"
)

var (
	// ErrInvalidID error when Session ID is invalid
	ErrInvalidID = errors.New("Invalid Session ID")
	// ErrInvalidAccessTime error when the Last Accessed Time is invalid
	ErrInvalidAccessTime = errors.New("Invalid Last Accessed Time")
)

// Session struct stored in MongoDB
type Session struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	Data         string        `bson:"data"`
	LastAccessed *time.Time    `bson:"lastAccessed"`
}

// MongoStore struct contains options and variables to interact with session settings
type MongoStore struct {
	Codecs         []securecookie.Codec
	Options        *sessions.Options
	Token          TokenGetSeter
	autoUpdateTime bool
	collection     string
	dbSession      *mgo.Session
}

// New returns a new MongoStore.
// Set ensureTTL to true let the database auto-remove expired object by maxAge.
// This is using *mgo.Session instead of *.mgo.Collection because if the database goes offline and then
// comes back onlne we will need the session to refresh the database connection.
// autoUpdateAccessTime - When true the sessions access time will be updated upon every access, even read operations.
// If false it will only update upon save, or manual intervention within the clients code.
func New(s *mgo.Session, collectionName string, options *sessions.Options, ensureTTL bool, autoUpdateAccessTime bool, keyPairs ...[]byte) *MongoStore {

	store := &MongoStore{
		Codecs:         securecookie.CodecsFromPairs(keyPairs...),
		Options:        options,
		Token:          &CookieToken{},
		autoUpdateTime: autoUpdateAccessTime,
		dbSession:      s,
		collection:     collectionName,
	}

	dbSess := s.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(store.collection)

	if ensureTTL {
		c.EnsureIndex(mgo.Index{
			Key:         []string{"lastAccessed"},
			Background:  true,
			Sparse:      true,
			ExpireAfter: time.Duration(options.MaxAge) * time.Second,
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

	var val string
	var err error
	var errToken error

	if val, errToken = m.Token.GetToken(r, name); errToken != nil {
		goto done
	}

	if err = securecookie.DecodeMulti(name, val, &session.ID, m.Codecs...); err != nil {
		goto done
	}

	if err = m.load(session); err != nil {
		goto done
	}

	session.IsNew = false

done:
	return session, err
}

// Save saves all sessions registered for the current request.
func (m *MongoStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {

	var err error

	if session.Options.MaxAge < 0 {

		if err = m.delete(session); err != nil {
			return err
		}

		m.Token.SetToken(w, session.Name(), "", session.Options)
		return nil
	}

	if session.ID == "" {
		session.ID = bson.NewObjectId().Hex()
	}

	if err = m.upsert(session); err != nil {
		return err
	}

	var encoded string

	if encoded, err = securecookie.EncodeMulti(session.Name(), session.ID, m.Codecs...); err != nil {
		return err
	}

	m.Token.SetToken(w, session.Name(), encoded, session.Options)

	return err
}

func (m *MongoStore) load(session *sessions.Session) error {

	if !bson.IsObjectIdHex(session.ID) {
		return ErrInvalidID
	}

	dbSess := m.dbSession.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(m.collection)

	var s *Session
	var err error

	if err = c.FindId(bson.ObjectIdHex(session.ID)).One(&s); err != nil {
		return err
	}

	if m.autoUpdateTime {

		accessed := time.Now().UTC()
		s.LastAccessed = &accessed

		if err = c.UpdateId(s.ID, s); err != nil {
			return err
		}
	}

	if err = securecookie.DecodeMulti(session.Name(), s.Data, &session.Values, m.Codecs...); err != nil {
		return err
	}

	return nil
}

func (m *MongoStore) upsert(session *sessions.Session) error {

	if !bson.IsObjectIdHex(session.ID) {
		return ErrInvalidID
	}

	var accessed time.Time

	if val, ok := session.Values[lastAccessedField]; ok {

		accessed, ok = val.(time.Time)

		if !ok {
			return ErrInvalidAccessTime
		}
	} else {
		accessed = time.Now().UTC()
	}

	var encoded string
	var err error

	if encoded, err = securecookie.EncodeMulti(session.Name(), session.Values, m.Codecs...); err != nil {
		return err
	}

	s := Session{
		ID:           bson.ObjectIdHex(session.ID),
		Data:         encoded,
		LastAccessed: &accessed,
	}

	dbSess := m.dbSession.Copy()
	defer dbSess.Close()

	c := dbSess.DB("").C(m.collection)

	if _, err = c.UpsertId(s.ID, &s); err != nil {
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
