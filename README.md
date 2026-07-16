# dmon — Interactive Docker Monitor

A terminal UI for monitoring Docker containers and restarting them safely, with
live CPU / memory / health metrics and one-key logs and shell access.

## Install

One line — installs the latest release for your OS/arch into `~/.local` (no sudo, no Go):

```bash
curl -fsSL https://github.com/ElyessBenSassi/dmon-docker-MadeEz/releases/latest/download/install.sh | bash
```

Pin a specific version:

```bash
curl -fsSL https://github.com/ElyessBenSassi/dmon-docker-MadeEz/releases/download/v0.2.0/install.sh | DMON_VERSION=v0.2.0 bash
```

The installer places the binary at `~/.local/share/dmon/<version>/dmon` and links
`~/.local/bin/dmon` to it. Make sure `~/.local/bin` is on your `PATH`.

**Requires:** a running Docker daemon. Linux and macOS, amd64 and arm64.

## Usage

Run from a directory containing a Docker Compose file, or point at one with `-f`:

```bash
dmon
dmon -f /path/to/docker-compose.yml
```

| Key | Action |
|-----|--------|
| `↑` / `k` · `↓` / `j` | Move selection |
| `r` | Restart selected container |
| `l` | View logs (last 100 lines) |
| `e` | Open a shell in the container (`bash` → `sh`) |
| `R` | Refresh now |
| `q` | Quit |

## Uninstall

```bash
rm -f ~/.local/bin/dmon && rm -rf ~/.local/share/dmon
```
