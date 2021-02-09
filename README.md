# nnBB (work in progress)

nnBB is a Web 1.0-style forum that progressively enhances into an [SSE][1] powered
live chat. It is a practice project, not intended for actual use.

[1]: https://en.wikipedia.org/wiki/Server-sent_events

The unusual thing about nnBB is that it has **no JavaScript code** of its own:
instead, server-rendered HTML is progressively enhanced with [intercooler][2].
This is because, firstly, I didn't want to build a frontend, and secondly,
I was intrigued by intercooler's [approach][3] and wanted to see how far I can
take it. The result has less polished UX than would be possible with a proper
frontend, but is much simpler to build and also completely usable without
JavaScript.

[2]: http://intercoolerjs.org/
[3]: http://intercoolerjs.org/docs.html#philosophy


## How to run

You need Go 1.16+ and a MongoDB replica set. Here's a quick way to spin up
a single-node MongoDB replica set on `localhost:27017` (without root):

    mongod --dbpath /some/empty/dir --replSet nnbb
    mongo --eval 'rs.initiate({_id: "nnbb", members: [{_id: 0, host: "localhost:27017"}]})'
    
Initialize the database with some fake data:

    go run github.com/vfaronov/nnbb/cmd/nnbbtool -init-db -insert-fake 100
    
Start the Web server:

    go run github.com/vfaronov/nnbb/cmd/nnbb -key mysecret
    
Then go to [`localhost:10242/rooms/`](http://localhost:10242/rooms/).
Or run a herd of test bots:

    go run github.com/vfaronov/nnbb/cmd/testbot
    
See also `-help` for each command.


## To Do

* more tests
* metrics
* CSRF protection
* paging in room list
* Markdown support
* changing room title
* deleting posts
* search
* watching rooms for unread posts
* user profile with recent posts
* lots more; see "TODO" in code
