# history - Connection History

Track and analyze your SSH connection history.

## Usage

```bash
lissh history [command]
```

## Commands

### list

View connection history.

```bash
lissh history list
lissh history list --limit=50      # Limit results
lissh history list --host=<id>     # Filter by host
```

### stats

Show connection statistics.

```bash
lissh history stats
```

Displays:
- Total connections
- Total time connected
- Most connected hosts
- Average session duration

### clear

Clear history for a host or all hosts.

```bash
lissh history clear <host-id>
lissh history clear --all          # Clear all history
lissh history clear --dry-run      # Preview
```

## How Tracking Works

When you connect to a host using `lissh <alias>` or `lissh hosts connect`, lissh automatically:

1. Starts a session when SSH begins
2. Ends the session when SSH exits
3. Records the duration

This lets you see:
- How often you connect to each host
- Total time spent on each host
- Last connection date

## Session Tracking

Sessions are tracked automatically when using lissh to connect. Manual SSH commands (`ssh user@host`) outside of lissh won't be tracked.

To disable tracking, use SSH directly without lissh.
