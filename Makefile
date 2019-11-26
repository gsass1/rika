BIN=rika

.PHONY: all
all: build

.PHONY: build
build:
	go build -o $(BIN)

.PHONY: test
test:
	go test

.PHONY: clean
clean:
	rm -f $(BIN)
