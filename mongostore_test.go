package mongostore

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	. "gopkg.in/bluesuncorp/assert.v1"
	"gopkg.in/mgo.v2"
)

// NOTES:
// - Run "go test" to run tests
// - Run "gocov test | gocov report" to report on test converage by file
// - Run "gocov test | gocov annotate -" to report on all code and functions, those ,marked with "MISS" were never called
//
// or
//
// -- may be a good idea to change to output path to somewherelike /tmp
// go test -coverprofile cover.out && go tool cover -html=cover.out -o cover.html
//

const (
	mySecretKeyString = "ISwearByMyPrettyFloralBonnetIWillEndYou"
	mySessionKey      = "session-key"
	myCustomKey       = "mycustomkey"
	url               = "http://localhost:8080"
	DatabaseName      = "session-db"
)

var (
	dbSession    *mgo.Session
	dbConnString = "127.0.0.1:27017"
)

type FlashMessage struct {
	Type    int
	Message string
}

func TestMain(m *testing.M) {

	// setup
	gob.Register(FlashMessage{})

	dbInfo := &mgo.DialInfo{
		Addrs:    []string{dbConnString},
		Timeout:  60 * time.Second,
		Database: DatabaseName,
	}

	// Create a session which maintains a pool of socket connections to our MongoDB
	dbSession, _ = mgo.DialWithInfo(dbInfo)

	dbSession.SetMode(mgo.Monotonic, true)

	dbSess := dbSession.Copy()
	defer dbSess.Close()

	database := dbSess.DB("")

	names, _ := database.CollectionNames()

	for _, val := range names {
		_ = database.C(val).DropCollection()
	}

	os.Exit(m.Run())

	// teardown
}

func TestMongoStoreCoreFuntionality(t *testing.T) {

	options := &sessions.Options{
		MaxAge: 3600,
		Path:   "/",
	}
	store := New(dbSession, "sessions", options, true, []byte(mySecretKeyString))

	r, _ := http.NewRequest("GET", url, nil)
	res := httptest.NewRecorder()

	// Get a session.
	session, err := store.GetAndUpdateAccessTime(r, res, mySessionKey)
	Equal(t, err, nil)

	// Attempt to retrieve flash messages
	flashMessages := session.Flashes()
	Equal(t, len(flashMessages), 0)

	// Add flash messages
	session.AddFlash("foo")
	session.AddFlash("bar")

	// Add Flash with a custom key
	session.AddFlash("baz", myCustomKey)

	// Save.
	err = sessions.Save(r, res)
	Equal(t, err, nil)

	header := res.Header()
	cookies, ok := header["Set-Cookie"]
	Equal(t, ok, true)
	Equal(t, len(cookies), 1)

	r, _ = http.NewRequest("GET", url, nil)
	r.Header.Add("Cookie", cookies[0])
	res = httptest.NewRecorder()

	// Get session.
	session, err = store.Get(r, mySessionKey)
	Equal(t, err, nil)

	flashMessages = session.Flashes()
	Equal(t, len(flashMessages), 2)
	Equal(t, flashMessages[0], "foo")
	Equal(t, flashMessages[1], "bar")

	flashMessages = session.Flashes()
	Equal(t, len(flashMessages), 0)

	// test custom key
	flashMessages = session.Flashes(myCustomKey)
	Equal(t, len(flashMessages), 1)
	Equal(t, flashMessages[0], "baz")

	flashMessages = session.Flashes(myCustomKey)
	Equal(t, len(flashMessages), 0)

	session.Options.MaxAge = -1

	err = sessions.Save(r, res)
	Equal(t, err, nil)

	r, _ = http.NewRequest("GET", url, nil)
	res = httptest.NewRecorder()

	session, err = store.Get(r, mySessionKey)
	Equal(t, err, nil)

	flashMessages = session.Flashes()
	Equal(t, len(flashMessages), 0)

	session.AddFlash(&FlashMessage{42, "foo"})

	err = sessions.Save(r, res)
	Equal(t, err, nil)

	header = res.Header()
	cookies, ok = header["Set-Cookie"]
	Equal(t, ok, true)
	Equal(t, len(cookies), 1)

	r, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	r.Header.Add("Cookie", cookies[0])
	res = httptest.NewRecorder()

	session, err = store.Get(r, mySessionKey)
	Equal(t, err, nil)

	flashMessages = session.Flashes()
	Equal(t, len(flashMessages), 1)

	customFlashMessage := flashMessages[0].(FlashMessage)
	Equal(t, customFlashMessage.Type, 42)
	Equal(t, customFlashMessage.Message, "foo")

	// Delete session.
	session.Options.MaxAge = -1

	err = sessions.Save(r, res)
	Equal(t, err, nil)

}
