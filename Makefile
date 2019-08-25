.PHONY: devel lint bin

devel: lint bin

lint:
	golangci-lint run

bin:
	go install github.com/vfaronov/chatter/cmd/chatter
