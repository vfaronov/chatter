.PHONY: devel lint bin

devel: bin lint

lint:
	golangci-lint run

bin:
	go install github.com/vfaronov/chatter/cmd/chatter
	go install github.com/vfaronov/chatter/cmd/chattertool
