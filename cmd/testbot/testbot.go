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

	"github.com/vfaronov/nnbb/testbot"
)

func main() {
	var (
		entryURL string
		n        int
		rate     float64
		seed     int64
	)
	flag.StringVar(&entryURL, "entry-url", "http://localhost:10242/signup/",
		"URL of the signup page of the nnBB instance to test")
	flag.IntVar(&n, "n", 100, "number of concurrent users to simulate")
	flag.Float64Var(&rate, "rate", 1.0,
		"speedup (> 1) / slowdown factor for each user")
	flag.Int64Var(&seed, "seed", 0, "random seed (0 to use current time)")
	flag.Parse()

	if seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(seed)
	}

	herd := testbot.NewHerd(entryURL, n, rate)
	log.Print("starting bot herd")
	go herd.Run()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Printf("terminating bot herd due to signal: %v", sig)
}
