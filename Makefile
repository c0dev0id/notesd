.PHONY: notesd notes notes-web clean test

BIN := $(CURDIR)/bin

notesd:
	mkdir -p $(BIN)
	$(MAKE) -C server BINDIR=$(BIN) build

notes:
	mkdir -p $(BIN)
	$(MAKE) -C notes-cli BINDIR=$(BIN) build

notes-web:
	cd web && npm run build

clean:
	$(MAKE) -C server BINDIR=$(BIN) clean
	$(MAKE) -C notes-cli BINDIR=$(BIN) clean
	rmdir --ignore-fail-on-non-empty $(BIN)

test:
	$(MAKE) -C server test
	$(MAKE) -C notes-cli test
