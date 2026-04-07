# actions - Remote Actions

Manage and execute predefined commands on your SSH hosts without manually connecting.

## Usage

```bash
lissh actions [command]
```

## Commands

### list

List all registered actions.

```bash
lissh actions list
```

### info

Show detailed information about an action including its command, bound hosts, and required variables.

```bash
lissh actions info <name>
```

### add

Create a new action.

```bash
lissh actions add <name> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--command` | The command template to execute (required) |
| `--description` | Description of what the action does |
| `--host-alias` | Comma-separated host aliases to bind (e.g., `sigma-dev,sigma-beta`) |

**Examples:**

```bash
# Create an action with bound hosts
lissh actions add logs \
  --description "Stream docker logs" \
  --command 'for id in $(docker ps --filter "name=${container}" --format "{{.ID}}"); do docker logs -f "$id" & done; wait' \
  --host-alias sigma-dev,sigma-beta

# Create an unbound action (run on any host with --host-alias)
lissh actions add check-memory \
  --description "Check server memory usage" \
  --command 'free -h'
```

### edit

Update an existing action.

```bash
lissh actions edit <name> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--command` | Updated command template |
| `--description` | Updated description |
| `--host-alias` | Comma-separated host aliases (replaces existing) |
| `--add-host` | Comma-separated host aliases to add (appends to existing) |

**Examples:**

```bash
# Update the command
lissh actions edit logs --command 'docker logs -f ${container}'

# Replace all bound hosts
lissh actions edit logs --host-alias sigma-dev,sigma-beta,production

# Add new hosts to existing bound hosts
lissh actions edit logs --add-host sigma-beta,new-server

# Update description
lissh actions edit logs --description "Stream container logs"
```

### delete

Delete an action.

```bash
lissh actions delete <name>
lissh actions delete <name> --confirm  # Skip confirmation
```

### run

Execute an action on its bound hosts (or a specific host).

```bash
lissh actions run <name> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--set` | Set variable values (e.g., `--set container=myapp`) |
| `--host-alias` | Run on specific host (must be bound to action) |
| `--alias` | Alias of bound host to run on (stricter than --host-alias) |

**Variable Substitution:**

Use `${variable_name}` in your command template. When running, provide values via `--set` or enter them interactively.

**Examples:**

```bash
# Run with variable provided via flag
lissh actions run logs --set container=sigma_merl

# Run (will prompt for ${container} value)
lissh actions run logs

# Run on a specific bound host
lissh actions run logs --alias sigma-beta --set container=api

# Run on unbound host (use --host-alias)
lissh actions run check-memory --host-alias webserver
```

## Variable Substitution

Actions support variable substitution using `${variable_name}` syntax:

```bash
# Command with variables
lissh actions add deploy \
  --command 'cd /opt/${app} && git pull && systemctl restart ${service}' \
  --host-alias production

# Run with variables
lissh actions run deploy --set app=myapp --set service=myapp.service
```

When a variable is not provided via `--set`, you will be prompted to enter a value interactively.

## Use Cases

### Docker Logs Streaming

```bash
lissh actions add docker-logs \
  --description "Stream docker container logs" \
  --command 'for id in $(docker ps --filter "name=${container}" --format "{{.ID}}"); do docker logs -f "$id" & done; wait' \
  --host-alias sigma-dev,sigma-beta

# Run
lissh actions run docker-logs --set container=sigma_merl
```

### System Monitoring

```bash
lissh actions add system-info \
  --description "Get system information" \
  --command 'echo "=== CPU ===" && nproc && echo "=== Memory ===" && free -h && echo "=== Disk ===" && df -h'

lissh actions run system-info
```

### Service Management

```bash
lissh actions add restart-app \
  --description "Restart application service" \
  --command 'sudo systemctl restart ${service}' \
  --host-alias production

lissh actions run restart-app --set service=myapp
```

## Multi-Host Execution

When an action is bound to multiple hosts, running the action executes it on each host sequentially:

```
→ [1/3] Host: 194.164.121.64
  Alias: sigma-beta
  ───────────────────────────────────────────────────────────
  (output from sigma-beta)

→ [2/3] Host: 82.165.205.201
  Alias: sigma-dev
  ───────────────────────────────────────────────────────────
  (output from sigma-dev)
```
