// Package config registers configuration flags that are common to various nnBB binaries.
// TODO: Need a better config system.
package config

import (
	"flag"
)

var (
	StoreURI string
	FakeData string
)

func WithStoreURI() {
	flag.StringVar(&StoreURI, "store-uri",
		"mongodb://localhost:27017/nnbb?replicaSet=nnbb",
		"connect to MongoDB at `URI` (must include DB name and replica set)")
}

func WithFakeData() {
	flag.StringVar(&FakeData, "fake-data", "",
		"for -insert-fake, use data from the given `FILE` instead of random "+
			"(see code comment on func NewFaker for details)")
}
