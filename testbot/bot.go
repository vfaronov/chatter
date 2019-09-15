// Package testbot implements fake users (bots) that sign up to Chatter, browse,
// submit and receive posts, roughly imitating actions of human users, and thereby
// serving as synthetic load for testing Chatter. They also do some checks
// on Chatter's functionality, thus doubling as an end-to-end test suite.
// Check failures are reported simply by panicking.
package testbot

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf/browser"
	"github.com/vfaronov/chatter/store"
	"gopkg.in/headzoo/surf.v1"
)

// Herd is a group of bots acting concurrently.
type Herd struct {
	signupURL string
	n         int
	rate      float64
}

// NewHerd returns an initialized Herd of n bots that will start at signupURL
// and proceed with further actions at the given relative rate (1.0 for "normal"
// rate as hardcoded into the bot).
func NewHerd(signupURL string, n int, rate float64) *Herd {
	return &Herd{signupURL, n, rate}
}

// Run executes the behavior of the Herd. This method never returns;
// a Herd cannot be stopped except by terminating the process.
func (h *Herd) Run() {
	for i := 0; i < h.n; i++ {
		go newBot(h, i).run()
	}
	select {} // TODO
}

type bot struct {
	herd     *Herd
	i        int
	name     string
	password string
	signedUp bool
	browser  *browser.Browser
}

func newBot(h *Herd, i int) *bot {
	browser := surf.NewBrowser()
	browser.SetUserAgent("testbot")
	return &bot{
		herd:     h,
		i:        i,
		name:     fmt.Sprintf("%v bot#%v", store.FakeUserName(), i),
		password: "12345",
		signedUp: false,
		browser:  browser,
	}
}

func (b *bot) delay() time.Duration {
	// Poisson process with a mean interarrival time of 10s/rate.
	return time.Duration(rand.ExpFloat64()*10000/b.herd.rate) * time.Millisecond
}

func (b *bot) logf(format string, v ...interface{}) {
	log.Printf("[%v] %v", b.name, fmt.Sprintf(format, v...))
}

func (b *bot) panicf(format string, v ...interface{}) {
	b.logf(format, v...)
	panic(fmt.Sprintf(format, v...))
}

func (b *bot) must(action string, err error) {
	if err != nil {
		b.panicf("failed to %v: %v", action, err)
	}
	if st := b.browser.StatusCode(); st != http.StatusOK {
		b.panicf("server responded: %v", st)
	}
}

func (b *bot) run() {
	b.logf("start running")
	for {
		dur := b.delay()
		b.logf("sleeping for %v", dur)
		time.Sleep(dur)
		switch url := b.browser.Url(); {
		case url == nil:
			b.browseToSignup()
		case strings.HasSuffix(url.Path, "/signup/"):
			b.signup()
		case strings.HasSuffix(url.Path, "/rooms/"):
			b.rooms()
		case strings.Contains(url.Path, "/rooms/"):
			b.room()
		default:
			b.panicf("don't know what to do on %v", url)
		}
	}
}

func (b *bot) browseToSignup() {
	b.logf("browsing to signup page")
	b.must("browse", b.browser.Open(b.herd.signupURL))
}

func (b *bot) signup() {
	if b.signedUp {
		b.logf("logging in")
	} else {
		b.logf("signing up")
	}
	form, err := b.browser.Form("form")
	b.must("find form", err)
	b.must("input name", form.Input("name", b.name))
	b.must("input password", form.Input("password", b.password))
	if b.signedUp {
		b.must("click log-in", form.ClickByValue("action", "log-in"))
	} else {
		b.must("click sign-up", form.ClickByValue("action", "sign-up"))
		b.signedUp = true
	}
}

func (b *bot) checkPage() {
	if name := b.browser.Find(".userinfo .author").Text(); name != b.name {
		b.panicf("logged in as %q but expected %q", name, b.name)
	}
}

func (b *bot) rooms() {
	b.logf("in rooms list = %v", b.browser.Url())
	b.checkPage()
	links := b.browser.Links()
	if rand.Intn(5) == 0 || len(links) == 0 {
		title := store.FakeRoomTitle()
		b.logf("creating a new room: %q", title)
		form, err := b.browser.Form("#newroom")
		b.must("find form", err)
		b.must("input title", form.Input("title", title))
		b.must("submit", form.Submit())
		return
	}
	link := pickLink(links)
	b.logf("following link to %q = %v", link.Text, link.Url())
	b.must("browse", b.browser.Open(link.Url().String()))
}

var (
	markExp    = regexp.MustCompile(`\$[0-9a-f]{12}`)
	eventIDExp = regexp.MustCompile(`(?m)^id: (.+)$`)
)

func (b *bot) room() {
	b.logf("in room %q = %v", b.browser.Title(), b.browser.Url())
	b.checkPage()

	// Open the updates stream concurrently, like a browser would do.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := b.updates(ctx)

	var (
		marks      []string
		expectOwn  string
		ownTimeout <-chan time.Time
	)
	for {
		dur := b.delay()
		b.logf("just chilling here for %v or until a new post", dur)
		select {
		case mark := <-updates:
			marks = append(marks, mark)
			if mark == expectOwn {
				ownTimeout = nil
			}

		case <-time.After(dur):
			if rand.Intn(2) == 0 {
				b.checkPosts(marks)
				b.logf("got bored, heading back to rooms list")
				b.must("browse", b.browser.Click("nav a"))
				return
			}
			mark, text := generatePost()
			b.logf("posting %q", mark)
			form, err := b.browser.Form("#newpost")
			b.must("find form", err)
			b.must("input text", form.Input("text", text))
			b.must("submit", form.Submit())
			expectOwn = mark
			ownTimeout = time.After(3 * time.Second)

		case <-ownTimeout:
			b.panicf("didn't get my own post %q from the stream", expectOwn)
		}
	}
}

func (b *bot) updates(ctx context.Context) <-chan string {
	u, ok := b.browser.Find("[ic-sse-src]").Attr("ic-sse-src")
	if !ok {
		b.panicf("no link to SSE here")
	}
	u, err := b.browser.ResolveStringUrl(u)
	b.must("resolve SSE URL", err)
	ch := make(chan string, 16)
	go b.streamUpdates(ctx, ch, u)
	return ch
}

func (b *bot) streamUpdates(ctx context.Context, ch chan<- string, streamURL string) {
	defer close(ch)
	var lastID string
	for {
		// TODO: insert a small delay (such as would be caused by a slow network)
		b.logf("opening stream %v (last event ID = %q)", streamURL, lastID)
		req, err := http.NewRequest("GET", streamURL, nil)
		b.must("create request", err)
		req.Header.Set("User-Agent", "testbot")
		req.Header.Set("Referer", b.browser.Url().String())
		if lastID != "" {
			req.Header.Set("Last-Event-Id", lastID)
		}
		for _, cookie := range b.browser.SiteCookies() {
			req.AddCookie(cookie)
		}
		resp, err := http.DefaultClient.Do(req.WithContext(ctx))
		if err != nil && ctx.Err() != nil {
			b.logf("stream aborted because already browsed away: %v", err)
			return
		}
		b.must("get response", err)
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			b.panicf("expected 200 OK, got %v", resp.Status)
		}

		sc := bufio.NewScanner(resp.Body)
		sc.Split(scanMessages)
		for sc.Scan() {
			msg := sc.Bytes()
			mark := string(markExp.Find(msg))
			if match := eventIDExp.FindSubmatch(msg); match != nil {
				lastID = string(match[1])
			}
			b.logf("got post %q from updates stream", mark)
			ch <- mark
		}
		resp.Body.Close()
		b.logf("event stream stopped at ID %v: %v", lastID, sc.Err())
		if ctx.Err() != nil { // canceled by us, not broken by server
			return
		}
		b.logf("resuming event stream")
	}
}

// scanMessages is a bufio.SplitFunc that splits text/event-stream
// into individual messages.
func scanMessages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// checkPosts checks that the expected posts (received from the updates stream),
// or at least some part of them (limited by page size), appear on the room page,
// in the same order, without intervening posts (preceding/following posts are OK).
// TODO: This is actually not guaranteed because posts with serial:5 and serial:6
// will be shown in that order on the room page but might arrive in the reverse
// order from MongoDB's change stream.
func (b *bot) checkPosts(expected []string) {
	if len(expected) == 0 {
		// Didn't stay on the room page for long enough to get any updates.
		return
	}
	b.logf("checking %v for expected posts: %v", b.browser.Url(), expected)
	// Not b.browser.Reload() because that would re-submit the last post.
	b.must("reload room page", b.browser.Open(b.browser.Url().String()))
	actual := b.browser.Find(".post").Map(func(i int, post *goquery.Selection) string {
		return markExp.FindString(post.Text())
	})
	i := len(expected) - 1
	j := len(actual) - 1
	// Skip any actual marks that may have been posted while we were reloading.
	for i >= 0 {
		if expected[i] == actual[j] {
			break
		}
		i--
	}
	// Walk expected and actual marks backwards, making sure they match.
	for i >= 0 && j >= 0 {
		if expected[i] != actual[j] {
			b.panicf("mismatch! actual posts: %v", actual)
		}
		i--
		j--
	}
}

// pickLink returns a random one of links, giving more weight to the earlier ones.
func pickLink(links []*browser.Link) *browser.Link {
	for i, link := range links {
		if rand.Intn(5) == 0 || i == len(links)-1 {
			return link
		}
	}
	return nil
}

func generatePost() (mark, text string) {
	// Mark each post with a unique string that will be easy to find in HTML later.
	b := make([]byte, 6)
	rand.Read(b)
	mark = fmt.Sprintf("$%x", b)
	text = fmt.Sprintf("%s %s", store.FakePostText(), mark)
	return
}
