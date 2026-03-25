.PHONY: notesd notes notes-web clean test

BINDIR ?= bin

notesd:
	mkdir -p $(BINDIR)
	$(MAKE) -C server BINDIR=$(BINDIR) build

notes:
	mkdir -p $(BINDIR)
	$(MAKE) -C notes-cli BINDIR=$(BINDIR) build

notes-web:
	cd web && npm run build

clean:
	$(MAKE) -C server BINDIR=$(BINDIR) clean
	$(MAKE) -C notes-cli BINDIR=$(BINDIR) clean
	rmdir $(BINDIR) 2>/dev/null || true

test:
	$(MAKE) -C server test
	$(MAKE) -C notes-cli test
