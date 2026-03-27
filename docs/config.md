# config - SSH Config Helpers

Manage SSH config entries for your hosts.

## Usage

```bash
lissh config [command]
```

## Commands

### generate

Generate SSH config entries for hosts.

```bash
lissh config generate
lissh config generate --host=<id>   # Specific host
lissh config generate --all          # All hosts
```

Outputs SSH config format:

```
Host webserver
    HostName 192.168.1.100
    User admin
    Port 22
    IdentityFile ~/.ssh/id_ed25519
```

### apply

Apply generated config to `~/.ssh/config` (backup created).

```bash
lissh config apply
lissh config apply --dry-run         # Preview first
```

### backup

Create a backup of your SSH config.

```bash
lissh config backup
```

Backups are stored in `~/.ssh/config.backup.<timestamp>`.

## SSH Config Integration

lissh can read hosts from your existing SSH config (`~/.ssh/config`) when you run:

```bash
lissh discover run
```

This means:
1. Hosts you add to SSH config manually
2. Custom settings like `IdentityFile`, `ProxyJump`, etc.

...will be picked up by lissh discovery.

## Workflow

Typical workflow with SSH config integration:

```bash
# 1. Add/edit host in SSH config manually or with lissh
nano ~/.ssh/config

# 2. Discover new hosts
lissh discover run

# 3. Add alias and notes
lissh hosts edit 16 --alias=webserver --notes="Production"

# 4. Associate SSH key
lissh hosts edit 16 --key-id=1

# 5. Connect
lissh webserver
```

## Config Format

lissh generates standard SSH config entries:

```
Host <alias>
    HostName <hostname-or-ip>
    User <user>
    Port <port>
    IdentityFile <key-path>
```

All fields are optional and only included when set.
