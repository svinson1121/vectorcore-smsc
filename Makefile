.PHONY: ui build test clean dev-ui all install uninstall

BINARY  = smsc
VERSION = 0.1.1a
PREFIX  = /opt/vectorcore
BINDIR  = $(PREFIX)/bin
ETCDIR  = $(PREFIX)/etc
LOGDIR  = $(PREFIX)/log
SYSTEMD = /lib/systemd/system/

all: ui build

# Build the React UI (required before `make build`)
ui:
	cd web && ([ -f package-lock.json ] && npm ci || npm install) && npm run build

# Build the Go binary (embeds web/dist if present)
build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY) ./cmd/smsc

# Run tests
test:
	go test ./...

# Start Vite dev server (proxies API to localhost:8080)
dev-ui:
	cd web && npm run dev

clean:
	rm -rf bin/ web/dist/

install: build
	install -d $(BINDIR)
	install -d $(ETCDIR)
	install -d $(LOGDIR)
	install -m755 bin/$(BINARY) $(BINDIR)/$(BINARY)
	@if [ ! -f $(ETCDIR)/smsc.yaml ]; then \
		install -m644 config.yaml $(ETCDIR)/smsc.yaml; \
	fi
	touch $(LOGDIR)/smsc.log
	chmod 644 $(LOGDIR)/smsc.log
	install -d /lib/systemd/system
	install -m644 systemd/vectorcore-smsc.service $(SYSTEMD)/vectorcore-smsc.service
	systemctl daemon-reload
	systemctl enable vectorcore-smsc
	systemctl start vectorcore-smsc

uninstall:
	systemctl stop vectorcore-smsc || true
	systemctl disable vectorcore-smsc || true
	rm -f $(BINDIR)/$(BINARY)
	rm -f $(SYSTEMD)/vectorcore-smsc.service
	systemctl daemon-reload
