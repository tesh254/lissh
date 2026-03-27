# keys - SSH Key Management

Manage SSH keys for authentication.

## Usage

```bash
lissh keys [command]
```

## Commands

### list

List all registered SSH keys.

```bash
lissh keys list
```

Shows private key path, public key path, type, and bits.

### scan

Scan `~/.ssh` for keys and register them.

```bash
lissh keys scan
```

Recognizes keys by naming patterns:
- `id_rsa`, `id_ed25519`, `id_ecdsa`, `id_dsa`
- `*_rsa`, `*_ed25519`, `*_ecdsa`, `*_dsa`

Skips non-key files like `known_hosts`, `config`, `authorized_keys`.

### info

Show detailed information about a key.

```bash
lissh keys info <id>
```

Shows path, type, size, comment, fingerprint, and associated hosts.

### create

Generate a new SSH keypair.

```bash
lissh keys create <filename> [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--type` | `ed25519` | Key type (rsa, ed25519) |
| `--bits` | 4096/256 | Key size (RSA default 4096, Ed25519 default 256) |
| `--comment` | "" | Comment for the key |

**Examples:**

```bash
# Create Ed25519 key (default)
lissh keys create mykey

# Create RSA key with custom bits
lissh keys create mykey --type=rsa --bits=8192

# Create with comment
lissh keys create deploy --comment="deploy@production"
```

### associate

Link a key to a host.

```bash
lissh keys associate <key-id> <host-id>
```

The key will be used when connecting to that host.

### remove

Remove a key from lissh (does not delete the key file).

```bash
lissh keys remove <id>
```

### delete

Permanently delete a key and its file.

```bash
lissh keys delete <id>
lissh keys delete <id> --dry-run     # Preview
lissh keys delete <id> --confirm      # Skip confirmation
```

**Warning:** This deletes both the private and public key files.

## Key Properties

Each key has:

- **ID** - Unique identifier
- **Name** - Filename (e.g., `id_ed25519`)
- **Path** - Private key location
- **Public Key Path** - Public key location (`<path>.pub`)
- **Type** - Key algorithm (rsa, ed25519, etc.)
- **Bits** - Key size
- **Comment** - Key comment (usually email or identifier)
- **Fingerprint** - SSH key fingerprint

## Using Keys with Hosts

To use a specific key when connecting, associate it with a host:

```bash
lissh keys associate 1 16
```

Or when editing a host:

```bash
lissh hosts edit 16 --key-id=1
```
