# hosts - Host Management

List, search, edit, and connect to your SSH hosts.

## Usage

```bash
lissh hosts [command]
```

## Commands

### list

List all known hosts.

```bash
lissh hosts list
lissh hosts list --all      # Include inactive hosts
```

### search

Search hosts by hostname, alias, IP, notes, or username.

```bash
lissh hosts search web
```

### info

Show detailed information about a host.

```bash
lissh hosts info <id>
```

### edit

Update a host's properties.

```bash
lissh hosts edit <id> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--alias` | Set a friendly alias |
| `--notes` | Add notes or labels |
| `--key-id` | Associate an SSH key |
| `--port` | Set custom port |
| `--user` | Set SSH user |

**Examples:**

```bash
# Set an alias
lissh hosts edit 16 --alias=webserver

# Add notes
lissh hosts edit 16 --notes="Production web server"

# Associate an SSH key
lissh hosts edit 16 --key-id=1

# Change port
lissh hosts edit 16 --port=2222

# Set SSH user
lissh hosts edit 16 --user=ubuntu
```

### connect

Connect to a host (also available as `lissh <alias>` at root level).

```bash
lissh hosts connect <id>
```

### inactivate

Mark a host as inactive (won't appear in lists).

```bash
lissh hosts inactivate <id>
```

### remove

Remove a host and its connection history.

```bash
lissh hosts remove <id>
lissh hosts remove <id> --dry-run  # Preview
```

## Direct Connection

Instead of using `hosts connect`, you can connect directly:

```bash
# By alias or hostname
lissh webserver
lissh 192.168.1.100

# By ID
lissh -i 16
```

## Host Properties

Each host has:

- **ID** - Unique identifier
- **Hostname** - Original hostname from known_hosts or SSH config
- **Alias** - Optional friendly name
- **IP Address** - Known IP (if discovered)
- **User** - Username for SSH (from SSH config or history)
- **Port** - SSH port (default: 22)
- **Notes** - Freeform notes/labels
- **SSH Key** - Associated SSH key for connection
- **Source** - Where it was discovered from
- **Status** - Active or inactive
