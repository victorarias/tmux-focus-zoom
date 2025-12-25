#!/usr/bin/env bash
# TPM plugin entry point for tmux-focus-zoom

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$CURRENT_DIR/tmux-focus-zoom"

# Build if binary doesn't exist
if [[ ! -x "$BINARY" ]]; then
    if command -v go &>/dev/null; then
        (cd "$CURRENT_DIR" && go build -o tmux-focus-zoom) || {
            tmux display-message "tmux-focus-zoom: Go build failed"
            exit 1
        }
    else
        tmux display-message "tmux-focus-zoom: Go required to build. Run 'make' manually."
        exit 1
    fi
fi

# Get user-configurable options with defaults
get_tmux_option() {
    local option="$1"
    local default="$2"
    local value
    value=$(tmux show-option -gqv "$option")
    echo "${value:-$default}"
}

# Keybinding for toggle (default: g)
toggle_key=$(get_tmux_option "@focus-zoom-key" "g")

# Set up keybinding
tmux bind-key "$toggle_key" run-shell "$BINARY toggle"

# Set up after-select-pane hook (more reliable than pane-focus-in)
# Use index [100] to avoid conflicts with other plugins
tmux set-hook -g 'after-select-pane[100]' "run-shell -b '$BINARY apply'"
