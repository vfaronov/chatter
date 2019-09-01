// Package config registers configuration flags that are common
// to all Chatter binaries.
package config

import (
	"flag"
)

var (
	StoreURI string
	WebAddr  string
)

func init() {
	flag.StringVar(&StoreURI, "store-uri",
		"mongodb://localhost:27017/chatter?replicaSet=chatter",
		"storage MongoDB connection string (must include DB name and replica set)")
	flag.StringVar(&WebAddr, "web-addr", "localhost:10242",
		"address for the Web server to listen on")
}
