# Nitpick

Hi!

I totally wanted a retro feel for browsing HN(https://news.ycombinator.com) over SSH, and it's basically all text anyway, so I had AI help me write a little TUI client for HN.

Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Browse all HN story types: Top, New, Ask, Show, Jobs.
- Threaded comment viewing with collapsible trees
- Login with your HN account (session persists across restarts)
- Upvote, reply, and submit stories
- Background notifications for replies to your comments
- Algolia-powered search
- Local SQLite cache for fast, offline-friendly browsing
- Vim-style keybindings

## Install

Requires Go 1.24+.

```bash
go install github.com/fragmede/nitpick@latest
```

Or build from source:

```bash
git clone https://github.com/fragmede/nitpick.git
cd nitpick
go build -o nitpick .
./nitpick
```

## Keybindings

### Navigation

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `Enter` | Open story or comment thread |
| `q` | Go back to list / quit |
| `Esc` | Go back one level |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+D` / `Ctrl+U` | Page down / up |
| `/` | Filter / search |
| `r` | Refresh |
| `Ctrl+R` | Hard refresh (bust cache) |

### Tabs

| Key | Action |
|---|---|
| `1`-`8` | Jump to tab (Top, New, Threads, Past, Comments, Ask, Show, Jobs) |
| `Tab` / `Shift+Tab` | Cycle through tabs |

### Comments

| Key | Action |
|---|---|
| `Space` | Collapse / expand comment tree |
| `p` / `[` | Jump to parent comment |
| `]` | Jump to next sibling |
| `u` | Upvote (requires login) |
| `r` | Reply (requires login) |
| `e` | Edit own comment (within 2hr window) |

### Actions

| Key | Action |
|---|---|
| `L` | Login |
| `s` | Submit a story |
| `n` | View notifications |
| `P` | View user profile |
| `o` | Open URL in browser |

## Configuration

Data is stored in `~/.config/nitpick/`:

| File | Purpose |
|---|---|
| `cache.db` | SQLite cache for stories, comments, and users |
| `session.json` | Persisted login session |

## License

[GPLv3](LICENSE)
