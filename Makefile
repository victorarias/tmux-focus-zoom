BINARY := tmux-focus-zoom
PREFIX ?= ~/.local
BINDIR := $(PREFIX)/bin
TMUX_PLUGINS := ~/.tmux/plugins/tmux-focus-zoom

.PHONY: all build test clean install uninstall install-tpm

all: build

build:
	go build -o $(BINARY)

test:
	go test -v ./...

clean:
	rm -f $(BINARY)

# Install binary to ~/.local/bin (standalone)
install: build
	mkdir -p $(BINDIR)
	cp $(BINARY) $(BINDIR)/
	@echo "Installed to $(BINDIR)/$(BINARY)"
	@echo ""
	@echo "Add to your tmux.conf:"
	@echo '  bind g run-shell "$(BINDIR)/$(BINARY) toggle"'
	@echo '  set-hook -g pane-focus-in[100] "run-shell -b '\''$(BINDIR)/$(BINARY) apply'\''"'

uninstall:
	rm -f $(BINDIR)/$(BINARY)

# Install as TPM plugin (symlink this repo)
install-tpm: build
	mkdir -p ~/.tmux/plugins
	ln -sf $(PWD) $(TMUX_PLUGINS)
	@echo "Installed as TPM plugin"
	@echo ""
	@echo "Add to your tmux.conf:"
	@echo "  set -g @plugin 'victorarias/tmux-focus-zoom'"
	@echo ""
	@echo "Then run: tmux source ~/.tmux.conf && prefix + I"
