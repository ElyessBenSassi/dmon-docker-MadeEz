# dmon — Interactive Docker Monitor

A terminal UI for monitoring Docker containers and restarting them safely, with
live CPU / memory / health metrics and one-key logs and shell access. Run it
anywhere to watch every container, or from a compose directory to group and
control that project's services.

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

Run dmon anywhere to monitor every container on the host:

```bash
dmon
```

Run it from a directory containing a Docker Compose file (or point at one with
`-f`) and dmon groups and highlights that project's services, and enables `u` to
bring the project up:

```bash
dmon -f /path/to/docker-compose.yml
```

| Key | Action |
|-----|--------|
| `↑` / `k` · `↓` / `j` | Move selection |
| `r` | Restart selected container |
| `l` | View logs (last 100 lines) |
| `e` | Open a shell in the container (`bash` → `sh`) |
| `u` | Compose up (compose project only): prompts for an optional profile, runs `docker compose up -d` |
| `R` | Refresh now |
| `q` | Quit |

When a compose project is detected, its containers are grouped under a
`compose · <project>` section, separate from `other containers`.

## Uninstall

```bash
rm -f ~/.local/bin/dmon && rm -rf ~/.local/share/dmon
```
