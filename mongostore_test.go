package mongostore_test

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"gopkg.in/bluesuncorp/mongo-session-store.v3"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
)

const (
	mySecretKeyString = "ISwearByMyPrettyFloralBonnetIWillEndYou"
	mySessionKey      = "session-key"
	myCustomKey       = "mycustomkey"
	url               = "http://localhost:8080"
	DatabaseName      = "session-db"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

type FlashMessage struct {
	Type    int
	Message string
}

var dbSession *mgo.Session

func (s *MySuite) SetUpSuite(c *C) {

	gob.Register(FlashMessage{})

	dbInfo := &mgo.DialInfo{
		Addrs:    []string{"127.0.0.1:27017"},
		Timeout:  60 * time.Second,
		Database: DatabaseName,
	}

	var err error
	// Create a session which maintains a pool of socket connections to our MongoDB
	dbSession, err = mgo.DialWithInfo(dbInfo)
	c.Assert(err, IsNil)

	dbSession.SetMode(mgo.Monotonic, true)

	dbSess := dbSession.Copy()
	defer dbSess.Close()

	database := dbSess.DB("")

	names, err := database.CollectionNames()
	c.Assert(err, IsNil)

	for _, val := range names {
		_ = database.C(val).DropCollection()
	}

}

func (ms *MySuite) TestMongoStoreCoreFuntionality(c *C) {

	options := &sessions.Options{
		MaxAge: 3600,
		Path:   "/",
	}
	store := mongostore.New(dbSession, "sessions", options, true, true, []byte(mySecretKeyString))

	r, _ := http.NewRequest("GET", url, nil)
	res := httptest.NewRecorder()

	// Get a session.
	session, err := store.Get(r, mySessionKey)
	c.Assert(err, IsNil)

	// Attempt to retrieve flash messages
	flashMessages := session.Flashes()
	c.Assert(len(flashMessages), Equals, 0)

	// Add flash messages
	session.AddFlash("foo")
	session.AddFlash("bar")

	// Add Flash with a custom key
	session.AddFlash("baz", myCustomKey)

	// Save.
	err = sessions.Save(r, res)
	c.Assert(err, IsNil)

	header := res.Header()
	cookies, ok := header["Set-Cookie"]
	c.Assert(ok, Equals, true)
	c.Assert(len(cookies), Equals, 1)

	r, _ = http.NewRequest("GET", url, nil)
	r.Header.Add("Cookie", cookies[0])
	res = httptest.NewRecorder()

	// Get session.
	session, err = store.Get(r, mySessionKey)
	c.Assert(err, IsNil)

	flashMessages = session.Flashes()
	c.Assert(len(flashMessages), Equals, 2)
	c.Assert(flashMessages[0], Equals, "foo")
	c.Assert(flashMessages[1], Equals, "bar")

	flashMessages = session.Flashes()
	c.Assert(len(flashMessages), Equals, 0)

	// test custom key
	flashMessages = session.Flashes(myCustomKey)
	c.Assert(len(flashMessages), Equals, 1)
	c.Assert(flashMessages[0], Equals, "baz")

	flashMessages = session.Flashes(myCustomKey)
	c.Assert(len(flashMessages), Equals, 0)

	session.Options.MaxAge = -1

	err = sessions.Save(r, res)
	c.Assert(err, IsNil)

	r, _ = http.NewRequest("GET", url, nil)
	res = httptest.NewRecorder()

	session, err = store.Get(r, mySessionKey)
	c.Assert(err, IsNil)

	flashMessages = session.Flashes()
	c.Assert(len(flashMessages), Equals, 0)

	session.AddFlash(&FlashMessage{42, "foo"})

	err = sessions.Save(r, res)
	c.Assert(err, IsNil)

	header = res.Header()
	cookies, ok = header["Set-Cookie"]
	c.Assert(ok, Equals, true)
	c.Assert(len(cookies), Equals, 1)

	r, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	r.Header.Add("Cookie", cookies[0])
	res = httptest.NewRecorder()

	session, err = store.Get(r, mySessionKey)
	c.Assert(err, IsNil)

	flashMessages = session.Flashes()
	c.Assert(len(flashMessages), Equals, 1)

	customFlashMessage := flashMessages[0].(FlashMessage)
	c.Assert(customFlashMessage.Type, Equals, 42)
	c.Assert(customFlashMessage.Message, Equals, "foo")

	// Delete session.
	session.Options.MaxAge = -1

	err = sessions.Save(r, res)
	c.Assert(err, IsNil)
}
