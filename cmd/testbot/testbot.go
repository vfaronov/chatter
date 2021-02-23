// testbot runs a herd of test bots against an nnBB instance.
package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vfaronov/nnbb/config"
	"github.com/vfaronov/nnbb/store"
	"github.com/vfaronov/nnbb/testbot"
)

func main() {
	config.WithFakeData()
	var entryURL string
	flag.StringVar(&entryURL, "entry-url", "http://localhost:10242/signup/",
		"URL of the signup page of the nnBB instance to test")
	var n int
	flag.IntVar(&n, "n", 100,
		"number of concurrent users to simulate")
	var rate float64
	flag.Float64Var(&rate, "rate", 1.0,
		"speedup (> 1) / slowdown factor for each user")
	var seed int64
	flag.Int64Var(&seed, "seed", 0,
		"random seed (0 to use current time)")
	flag.Parse()

	if seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(seed)
	}

	faker, err := store.NewFaker(config.FakeData)
	if err != nil {
		log.Fatalf("failed to load fake data: %v", faker)
	}

	herd := testbot.NewHerd(faker, entryURL, n, rate)
	log.Print("starting bot herd")
	go herd.Run()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Printf("terminating bot herd due to signal: %v", sig)
}
