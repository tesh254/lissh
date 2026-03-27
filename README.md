# lissh - SSH on a leash

<p align="left">
    <a href="https://github.com/wcrg/lissh/actions/workflows/ci.yml"><img src="https://github.com/wcrg/lissh/actions/workflows/ci.yml/badge.svg?branch=main&event=push" alt="CI Tasks for lissh"></a>
    <a href="https://goreportcard.com/report/github.com/wcrg/lissh"><img src="https://goreportcard.com/badge/github.com/wcrg/lissh" alt="Go Report Card"></a>
    <a href="https://pkg.go.dev/github.com/wcrg/lissh"><img src="https://pkg.go.dev/badge/www.github.com/wcrg/lissh" alt="PkgGoDev"></a>
</p>

**lissh** keeps your SSH hosts organized and on a leash. Quickly list and search hosts, assign friendly aliases, track your connection history, manage SSH keys, and tweak SSH settings - all without manually editing config files.

## Features

- **Host Discovery**: Automatically discover hosts from `known_hosts` and `~/.ssh/config`
- **Host Labeling**: Assign friendly aliases and notes to your servers
- **Connection History**: Track when you connected and how long sessions lasted
- **SSH Key Management**: Create, list, and associate keys with hosts
- **SSH Config Helpers**: Apply recommended settings with backups and diffs
- **SQLite Storage**: Local database for all your SSH inventory and history

## Installation

```bash
go install github.com/wcrg/lissh@latest
```

Or build from source:

```bash
git clone https://github.com/wcrg/lissh.git
cd lissh
go build -o lissh
```

## Quickstart

### 1. Discover your hosts

```bash
lissh discover run
```

This scans `known_hosts` and `~/.ssh/config` to find your SSH-accessible servers.

### 2. Label your hosts

```bash
lissh discover review
```

Step through each host and assign friendly aliases like "web-prod" or "db-primary".

### 3. Connect quickly

```bash
lissh hosts list
lissh hosts connect 1
```

Use aliases or IDs to jump into SSH sessions.

## Commands

### `lissh hosts`

Manage your SSH hosts.

```bash
lissh hosts list              # List all hosts
lissh hosts search <term>     # Search by name, alias, or notes
lissh hosts info <id>         # Show detailed host info
lissh hosts edit <id>         # Edit alias, notes, or key
lissh hosts connect <id>      # SSH into the host
lissh hosts inactivate <id>   # Mark as inactive (keeps history)
lissh hosts remove <id>       # Permanently remove
lissh hosts prune             # Remove all inactive hosts
```

### `lissh discover`

Discover and label hosts.

```bash
lissh discover run             # Scan for new hosts
lissh discover review          # Step through and label hosts
```

### `lissh keys`

Manage SSH keys.

```bash
lissh keys list               # List registered keys
lissh keys info <id>          # Show key details
lissh keys create <name>      # Create new keypair
lissh keys associate <key-id> <host-id>  # Link key to host
lissh keys remove <id>        # Remove from lissh (keeps file)
lissh keys delete <id>        # Permanently delete key
```

### `lissh history`

View connection history.

```bash
lissh history list             # Recent connections
lissh history host <id>       # History for specific host
lissh history clear           # Clear all history
```

### `lissh config`

SSH configuration helpers.

```bash
lissh config show             # Show current SSH config
lissh config diff             # Show recommended changes
lissh config apply           # Apply recommended settings
lissh config backup          # Backup SSH config
lissh config restore <file>  # Restore from backup
lissh config keepalive on     # Enable keepalive
lissh config keepalive off    # Disable keepalive
lissh config compression on   # Enable compression
lissh config controlmaster auto  # Enable ControlMaster
```

## Configuration

lissh stores its data in `~/.lissh/lissh.db`. You can specify a custom path with `--db-path`.

## Database Schema

The SQLite database contains:

- **hosts**: Host inventory with aliases, notes, and source
- **ssh_keys**: SSH key metadata
- **history**: Connection sessions with timestamps and duration
- **schema_migrations**: Migration tracking

## License

MIT
