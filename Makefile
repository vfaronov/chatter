.PHONY: check fakebuild lint bin nnbb nnbbtool testbot

check: fakebuild lint

fakebuild:
	go build -mod=readonly -o /dev/null github.com/vfaronov/nnbb/cmd/nnbb
	go build -mod=readonly -o /dev/null github.com/vfaronov/nnbb/cmd/nnbbtool
	go build -mod=readonly -o /dev/null github.com/vfaronov/nnbb/cmd/testbot

bin: nnbb nnbbtool testbot

nnbb:
	go build github.com/vfaronov/nnbb/cmd/nnbb

nnbbtool:
	go build github.com/vfaronov/nnbb/cmd/nnbbtool

testbot:
	go build github.com/vfaronov/nnbb/cmd/testbot

lint:
	golangci-lint run
