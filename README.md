# tmux-focus-zoom

Dynamic pane zooming for tmux - the focused pane automatically grows while others shrink proportionally.

Unlike tmux's built-in zoom (`prefix + z`) which hides other panes entirely, focus-zoom keeps all panes visible while giving more space to what you're working on.

## Features

- **Proportional resizing**: When you focus a pane, it grows to 65% (configurable) while others shrink proportionally
- **Layout preservation**: Toggle off to restore original layout exactly
- **Nested support**: Works with complex layouts (columns within columns, rows within rows)
- **Automatic**: Once enabled, zoom follows your focus as you switch panes

## Demo

```
Before (equal columns):          After focusing right pane:
┌────────┬────────┬────────┐    ┌────┬────┬────────────────┐
│        │        │        │    │    │    │                │
│  33%   │  33%   │  33%   │ -> │17% │17% │      65%       │
│        │        │        │    │    │    │                │
└────────┴────────┴────────┘    └────┴────┴────────────────┘
```

## Installation

### With TPM (recommended)

Add to your `~/.tmux.conf`:

```tmux
set -g @plugin 'victorarias/tmux-focus-zoom'
```

Then press `prefix + I` to install.

### Manual

```bash
# Clone and build
git clone https://github.com/victorarias/tmux-focus-zoom.git
cd tmux-focus-zoom
make install

# Add to ~/.tmux.conf
bind g run-shell "~/.local/bin/tmux-focus-zoom toggle"
set-hook -g pane-focus-in[100] "run-shell -b '~/.local/bin/tmux-focus-zoom apply'"
```

### From source (requires Go 1.23+)

```bash
go install github.com/victorarias/tmux-focus-zoom@latest
```

## Usage

| Key | Action |
|-----|--------|
| `prefix + g` | Toggle focus-zoom on/off |

When enabled:
- Moving focus to a pane automatically resizes it to 65% of the window
- Other panes shrink proportionally (not equally)
- Toggle off to restore the original layout

## Configuration

Add these to your `~/.tmux.conf` before the plugin line:

```tmux
# Zoom percentage (10-95, default: 65)
set -g @focus-zoom-percent 70

# Toggle keybinding (default: g)
set -g @focus-zoom-key g
```

## Status Bar Integration

Show zoom state in your status bar:

```tmux
set -g status-right "#(tmux-focus-zoom status) ..."
```

This displays:
- `󰍉 ON` when focus-zoom is active
- `󰍉 OFF` when disabled

## How It Works

1. **Snapshot**: When enabled, captures the current window layout
2. **Parse**: Converts tmux layout string into a tree structure
3. **Calculate**: Computes new sizes where focused pane gets 65%, others shrink proportionally
4. **Apply**: Rebuilds layout string and applies with `select-layout`
5. **Restore**: Toggle off restores the original snapshot exactly

This approach ensures ALL panes resize correctly, unlike `resize-pane` which only affects adjacent panes.

## Requirements

- tmux 3.0+
- Go 1.23+ (for building)

## License

GPL-3.0 - See [LICENSE](LICENSE) for details.
