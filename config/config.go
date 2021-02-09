// Package config registers configuration flags that are common
// to all nnBB binaries.
package config

import (
	"flag"
)

var (
	StoreURI string
)

func init() {
	flag.StringVar(&StoreURI, "store-uri",
		"mongodb://localhost:27017/nnbb?replicaSet=nnbb",
		"storage MongoDB connection string (must include DB name and replica set)")
}
