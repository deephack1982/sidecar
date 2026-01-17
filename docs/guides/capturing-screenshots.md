# Capturing Sidecar Screenshots

This guide explains how to capture screenshots of Sidecar for documentation purposes.

## Prerequisites

- `tmux` - for running sidecar in a detached session
- `termshot` - for rendering terminal output as PNG (`brew install homeport/tap/termshot`)
- `aha` (optional) - for HTML output (`brew install aha`)

## Terminal Size

For documentation screenshots, resize your terminal to approximately **120x45** characters before capturing. This produces screenshots that fit well in documentation without being too large.

## Agent-Controlled Screenshots (Recommended)

Use the helper script `scripts/tmux-screenshot.sh` with simple subcommands:

### Step 1: Start sidecar

```bash
./scripts/tmux-screenshot.sh start
```

This starts sidecar in a detached tmux session sized to your current terminal.

### Step 2: Attach and navigate

```bash
./scripts/tmux-screenshot.sh attach
```

Or directly: `tmux attach -t sidecar-screenshot`

Once attached:
1. Navigate to screens using number keys: **1=TD, 2=Git, 3=Files, 4=Conversations, 5=Worktrees**
2. Within a screen, use **j/k** (or arrow keys) to navigate up/down
3. Press **Enter** or **Space** to interact with items
4. Detach from tmux with **Ctrl+A D** (the tmux prefix in this session is Ctrl+A)

### Step 3: Capture the screenshot

```bash
./scripts/tmux-screenshot.sh capture sidecar-td
```

This captures the current view and:
1. Renders terminal output as PNG with proper fonts and colors (requires `termshot`)
2. Optionally creates HTML backup (if `aha` is installed)
3. Saves files to `docs/screenshots/`

### Step 4: Repeat or cleanup

Repeat steps 2-3 for additional screenshots, then:

```bash
./scripts/tmux-screenshot.sh stop
```

## Script Commands
|| Command | Description |
|---------|-------------|
| `start` | Start sidecar in a tmux session |
| `attach` | Attach to navigate (detach with Ctrl+A/B D) |
| `capture NAME` | Capture current view to `docs/screenshots/NAME.html` and `NAME.png` |
| `list` | List existing screenshots |
| `stop` | Quit sidecar and kill session |

## LLM Workflow

For AI agents, run `tmux attach -t sidecar-screenshot` in **interact mode** to navigate. The workflow:

1. `./scripts/tmux-screenshot.sh start`
2. `tmux attach -t sidecar-screenshot` (in interact mode) → navigate to screen using number keys → interact with content → `Ctrl+A D` to detach
3. `./scripts/tmux-screenshot.sh capture sidecar-{plugin}`
4. Repeat 2-3 for each plugin
5. `./scripts/tmux-screenshot.sh stop`

### Important: Tmux Setup for Agent Interaction

Before you can interact with sidecar via tmux, you must configure tmux to allow direct key input:

1. **Create or update `~/.tmux.conf`** with these settings:
   ```
   set -g mouse on
   set -g mode-keys vi
   unbind C-b
   set -g prefix C-a
   bind C-a send-prefix
   ```
   The critical settings are:
   - `set -g prefix C-a` - Sets the tmux prefix to Ctrl+A (used to detach with Ctrl+A D)
   - `set -g mode-keys vi` - Enables vi key bindings (j/k for navigation, etc.)

2. **Reload the config or restart tmux:**
   ```bash
   tmux source-file ~/.tmux.conf
   # or
   tmux kill-server  # kills all sessions
   ```
   After restarting tmux, the session will be ready for agent interaction.

3. **In interact mode:**
   - Press number keys to navigate screens (1-5)
   - Use `j/k` or arrow keys to scroll through content
   - Press `Enter` or `Space` to select/preview items
   - Press `Ctrl+A D` to detach (not `Ctrl+B D`)

**Screen navigation keys:**
- **1** = TD (task management)
- **2** = Git
- **3** = Files (file browser)
- **4** = Conversations
- **5** = Worktrees

**Within a screen:**
- **j/k** or arrow keys = navigate items
- **Enter/Space** = interact with selected item (select commit, preview file, etc.)
- **Ctrl+A D** = detach from tmux session

**Important for agents:** Tmux must be configured with `set -g prefix C-a` (see "Tmux Setup for Agent Interaction" above). Always use **Ctrl+A D** to detach from the tmux session.

## Why Interactive?

`tmux send-keys` doesn't reliably trigger sidecar's keybindings. Attaching and pressing keys directly in interact mode always works. This allows you to:
- Navigate to specific screens
- Select commits, files, or other items to display interesting content
- Capture the full interactive state of sidecar

The interact mode provides a live PTY interface where you can press keys in real-time, making it ideal for getting the UI into the exact state you want before capturing.

## Viewing Captures

```bash
./scripts/tmux-screenshot.sh list       # List screenshots
open docs/screenshots/sidecar-td.html   # View HTML in browser
open docs/screenshots/sidecar-td.png    # View PNG image
```

Both HTML and PNG files are retained as artifacts. The PNG provides pixel-perfect rendering for documentation, while the HTML preserves the original ANSI-to-HTML conversion for reference.
