# mongo-session-store
Gorilla's session store implementation using MongoDB

[![Build Status](https://travis-ci.org/bluesuncorp/mongo-session-store.svg?branch=v3)](https://travis-ci.org/bluesuncorp/mongo-session-store)
[![GoDoc](https://godoc.org/gopkg.in/bluesuncorp/mongo-session-store.v3?status.svg)](https://godoc.org/gopkg.in/bluesuncorp/mongo-session-store.v3)

Installation
============

Just use go get.

	go get gopkg.in/bluesuncorp/mongo-session-store.v3

or to update

	go get -u gopkg.in/bluesuncorp/mongo-session-store.v3

And then just import the package into your own code.

	import "gopkg.in/bluesuncorp/mongo-session-store.v3"

Usage
=====

Please see http://godoc.org/gopkg.in/bluesuncorp/mongo-session-store.v3 for detailed usage docs.

Contributing
============

There will be a development branch for each version of this package i.e. v1-development, please
make your pull requests against those branches.

If changes are breaking please create an issue, for discussion and create a pull request against
the highest development branch for example this package has a v1 and v1-development branch
however, there will also be a v2-development brach even though v2 doesn't exist yet.

License
=========
Distributed under MIT License, see license file in code for details.