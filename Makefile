.PHONY: notesd notes notes-web clean test

notesd:
	$(MAKE) -C server build

notes:
	$(MAKE) -C notes-cli build

notes-web:
	cd web && npm run build

clean:
	$(MAKE) -C server clean
	$(MAKE) -C notes-cli clean

test:
	$(MAKE) -C server test
	$(MAKE) -C notes-cli test
