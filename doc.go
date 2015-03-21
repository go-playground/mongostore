/*
Mongo Session Store follows the Gorilla Session implementation

Reason for creatio over using an existing library
	When the MongoDB database the sessions were being stored in was not reachable
	especially in the event of a database cycle other libraries would not
	restablish the database session; this library will.

*/

package store
