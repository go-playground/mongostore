/*
Package mongostore follows the Gorilla Session implementation

Reason for creation over using an existing library
	When the MongoDB database the sessions were being stored in was not reachable
	especially in the event of a database cycle other libraries would not
	restablish the database session; this library will.

Example Usage:

	func foo(w http.ResponseWriter, r *http.Request) {

        // Coonect to MongoDB
        dbSess, err := mgo.Dial("localhost")
        if err != nil {
            panic(err)
        }
        defer dbSess.Close()

        store := mongostore.New(dbSess, "sessions", 3600, true,
            []byte("secret-key"))

        // Get a session.
        session, err := store.Get(r, "session-key")
        if err != nil {
            log.Println(err.Error())
        }

        // Add a value.
        session.Values["foo"] = "bar"

        // Save.
        if err = sessions.Save(r, w); err != nil {
            log.Printf("Error saving session: %v", err)
        }

        fmt.Fprintln(w, "ok")
    }

Updating Cookie and MongoDB Expiry Times

    if you need your cookies to be rolling and update with every access and
    not just modifications you can call:

    // Get a session.
    session, err := store.GetAndUpdateAccessTime(r, w, "session-key")

    instead of

    // Get a session.
    session, err := store.Get(r, "session-key")

*/
package mongostore
