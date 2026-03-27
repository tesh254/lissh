# discover - Host Discovery

Discover SSH hosts from your system configuration files.

## Usage

```bash
lissh discover [command]
```

## Commands

### run

Scan known_hosts and SSH config to discover hosts.

```bash
lissh discover run
lissh discover run --dry-run  # Preview without adding
lissh discover run --all      # Show already known hosts too
```

### review

Interactively review and label discovered hosts.

```bash
lissh discover review
```

For each host you can:
- `[a]` Set an alias
- `[n]` Add notes
- `[k]` Associate an SSH key
- `[i]` Mark as inactive
- `[s]` Skip
- `[q]` Quit

### users

Infer usernames from shell history and update host records.

```bash
lissh discover users
lissh discover users --dry-run  # Preview without making changes
```

This scans `~/.bash_history` and `~/.zsh_history` for SSH commands like:

```
ssh user@hostname
ssh -l user hostname
ssh -p 2222 hostname
```

Matching hosts without a user set will be updated.

## Sources

Hosts can be discovered from:

- `~/.ssh/known_hosts` - Known hosts file
- `~/.ssh/config` - SSH config file (Host entries)

## Adding Hosts Manually

While discovery finds hosts automatically, you can also:

1. Edit `~/.ssh/config` and run `lissh discover run`
2. Direct SSH to a new host (it gets added to known_hosts)
3. Run discovery again
