include config.mk

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

.PHONY: install
install: all
	@mkdir -p ${DESTDIR}${PREFIX}/bin
	@cp -f rika ${DESTDIR}${PREFIX}/bin
	@chmod 755 ${DESTDIR}${PREFIX}/bin/rika

.PHONY: uninstall
uninstall:
	@rm -f ${DESTDIR}${PREFIX}/bin/sic
