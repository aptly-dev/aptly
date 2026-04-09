# GPG Keys Management

GPG keys are used by aptly to verify the authenticity of remote repository Release files when creating mirrors. This document describes the API endpoints for managing GPG keys in the aptly keyring.

## Overview

Aptly uses GNU Privacy Guard (GPG) to verify signed repository metadata. You must add the repository's GPG public key to aptly's keyring before creating mirrors that verify signatures.

Keys are stored in the aptly keyring (default: `trustedkeys.gpg`). You can have multiple keyrings and specify which one to use via the `Keyring` parameter.

## API Endpoints

### List GPG Keys

**GET /api/gpg/keys**

Lists all public GPG keys currently installed in the aptly keyring.

**Parameters:**
- `keyring` (query, optional): Keyring file to list keys from. Default: `trustedkeys.gpg`

**Response:**
```json
{
  "Keys": [
    {
      "KeyID": "8B48AD6246925553",
      "Fingerprint": "D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0",
      "Validity": "f",
      "UserIDs": ["John Doe <john@example.com>"],
      "CreatedAt": "1611864000"
    }
  ]
}
```

**Status Codes:**
- `200 OK`: Keys successfully retrieved
- `400 Bad Request`: GPG execution failed or invalid parameters

**Example:**
```bash
curl http://localhost:8080/api/gpg/keys
curl "http://localhost:8080/api/gpg/keys?keyring=custom.gpg"
```

---

### Add GPG Key

**POST /api/gpg/key**

Adds a GPG public key to the aptly keyring. Keys can be added in two ways:
1. Provide the ASCII-armored key directly
2. Provide a key server and key ID(s) to download from

**Request Body:**
```json
{
  "Keyring": "trustedkeys.gpg",
  "GpgKeyArmor": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n...",
  "Keyserver": "hkp://keyserver.ubuntu.com:80",
  "GpgKeyID": "8B48AD6246925553"
}
```

**Parameters:**
- `Keyring` (optional): Keyring file to add keys to. Default: `trustedkeys.gpg`
- `GpgKeyArmor` (optional): ASCII-armored GPG public key
- `Keyserver` (optional): Keyserver URL (e.g., `hkp://keyserver.ubuntu.com:80`)
- `GpgKeyID` (optional): Space-separated key IDs to download from keyserver

**Status Codes:**
- `200 OK`: Key successfully added
- `400 Bad Request`: Invalid parameters or GPG execution failed

**Example - From ASCII Key:**
```bash
curl -X POST http://localhost:8080/api/gpg/key \
  -H "Content-Type: application/json" \
  -d '{
    "GpgKeyArmor": "-----BEGIN PGP PUBLIC KEY BLOCK-----\nVersion: GnuPG v2\n...\n-----END PGP PUBLIC KEY BLOCK-----"
  }'
```

**Example - From Keyserver:**
```bash
curl -X POST http://localhost:8080/api/gpg/key \
  -H "Content-Type: application/json" \
  -d '{
    "Keyserver": "hkp://keyserver.ubuntu.com:80",
    "GpgKeyID": "8B48AD6246925553 A1B2C3D4E5F67890"
  }'
```

---

### Delete GPG Key

**DELETE /api/gpg/key**

Removes a GPG key from the aptly keyring.

**Request Body:**
```json
{
  "Keyring": "trustedkeys.gpg",
  "GpgKeyID": "8B48AD6246925553"
}
```

**Parameters:**
- `Keyring` (optional): Keyring file to delete from. Default: `trustedkeys.gpg`
- `GpgKeyID` (required): Key ID or fingerprint to delete

**Status Codes:**
- `200 OK`: Key successfully deleted
- `400 Bad Request`: Invalid parameters or GPG execution failed

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/gpg/key \
  -H "Content-Type: application/json" \
  -d '{
    "GpgKeyID": "8B48AD6246925553"
  }'
```

---

## Use Cases

### 1. Verify Downloaded Repository Metadata

Before creating a mirror from a signed repository, add the repository's GPG key:

```bash
# Add the key from a keyserver
curl -X POST http://localhost:8080/api/gpg/key \
  -H "Content-Type: application/json" \
  -d '{
    "Keyserver": "hkp://keyserver.ubuntu.com:80",
    "GpgKeyID": "EB9B46B91F2D3B7E"
  }'

# Now create a mirror with signature verification
# (signature verification configured in mirror settings)
```

### 2. Manage Multiple Keyrings

Aptly supports using different keyrings for different purposes. For example, one for Debian repositories and another for custom internal repositories:

```bash
# Add key to Debian keyring
curl -X POST http://localhost:8080/api/gpg/key \
  -H "Content-Type: application/json" \
  -d '{
    "Keyring": "debian-keys.gpg",
    "Keyserver": "hkp://keyserver.ubuntu.com:80",
    "GpgKeyID": "EB9B46B91F2D3B7E"
  }'

# List keys in Debian keyring
curl "http://localhost:8080/api/gpg/keys?keyring=debian-keys.gpg"
```

### 3. Remove Compromised Keys

If a GPG key is compromised, remove it from the keyring immediately:

```bash
curl -X DELETE http://localhost:8080/api/gpg/key \
  -H "Content-Type: application/json" \
  -d '{
    "GpgKeyID": "COMPROMISED_KEY_ID"
  }'
```

---

## Key Validity Values

Keys retrieved from `GET /api/gpg/keys` have a `Validity` field with the following possible values:

- `u` — Unknown validity
- `f` — Full trust
- `m` — Marginal trust
- `n` — Never trust
- `-` — Trust not set

The trust level is typically managed in your GPG configuration and does not affect aptly's ability to verify signatures.

---

## Troubleshooting

**"failed to list keys"**
- Check that the keyring file exists and is readable
- Verify GPG is installed and configured

**"unable to delete key: no public key"**
- The key might not exist in the keyring
- Verify the key ID is correct by listing keys first

**"invalid request body"**
- Ensure the JSON is properly formatted
- For POST requests, provide either `GpgKeyArmor` or (`Keyserver` + `GpgKeyID`)
- For DELETE requests, `GpgKeyID` is required
