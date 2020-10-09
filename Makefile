.PHONY: check fakebuild lint bin chatter chattertool chatterbot

check: fakebuild lint

fakebuild:
	go build -mod=readonly -o /dev/null github.com/vfaronov/chatter/cmd/chatter
	go build -mod=readonly -o /dev/null github.com/vfaronov/chatter/cmd/chattertool
	go build -mod=readonly -o /dev/null github.com/vfaronov/chatter/cmd/chatterbot

bin: chatter chattertool chatterbot

chatter:
	go build github.com/vfaronov/chatter/cmd/chatter
	rice append --exec=chatter github.com/vfaronov/chatter/web

chattertool:
	go build github.com/vfaronov/chatter/cmd/chattertool

chatterbot:
	go build github.com/vfaronov/chatter/cmd/chatterbot

lint:
	golangci-lint run
