# lissh - SSH on a Leash

A CLI tool for managing SSH hosts, aliases, keys, and connection history.

## Quick Start

```bash
# Connect to a host by alias or hostname
lissh webserver
lissh 192.168.1.100

# Connect to a host by ID
lissh -i 16

# List all hosts
lissh hosts list

# Discover hosts from known_hosts and SSH config
lissh discover run

# Scan for SSH keys
lissh keys scan
```

## Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--db-path` | | Path to lissh database | `~/.lissh/lissh.db` |
| `--id` | `-i` | Connect to host by ID | |

## Commands

- [hosts](./hosts.md) - List, search, edit, and connect to hosts
- [discover](./discover.md) - Discover hosts from known_hosts and SSH config
- [keys](./keys.md) - Manage SSH keys
- [history](./history.md) - View connection history
- [config](./config.md) - SSH config helpers

## Database Location

By default, lissh stores its SQLite database at `~/.lissh/lissh.db`. You can specify a different location with the `--db-path` flag:

```bash
lissh --db-path=/custom/path/lissh.db hosts list
```
