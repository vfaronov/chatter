.PHONY: devel lint bin chatter chattertool chatterbot

devel: bin lint

bin: chatter chattertool chatterbot

chatter:
	go install github.com/vfaronov/chatter/cmd/chatter

chattertool:
	go install github.com/vfaronov/chatter/cmd/chattertool

chatterbot:
	go install github.com/vfaronov/chatter/cmd/chatterbot

lint:
	golangci-lint run
