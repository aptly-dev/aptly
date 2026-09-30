# 10. Signing & Verification (`pgp/`)

aptly touches OpenPGP in two directions:

| Direction | Purpose | Interface | Where used |
|-----------|---------|-----------|------------|
| **Verify** (inbound) | Trust remote `Release`/`InRelease`, source `.dsc`, `.changes` uploads, installer `SHA256SUMS` | `pgp.Verifier` | `mirror create/update`, `repo add` (`.dsc`), `repo include` |
| **Sign** (outbound) | Sign the repositories aptly publishes | `pgp.Signer` | `publish snapshot/repo/switch/update`, API publish |

```go
type Signer interface {
    Init() error ; SetKey(keyRef string) ; SetKeyRing(keyring, secretKeyring string)
    SetPassphrase(passphrase, passphraseFile string) ; SetBatch(bool)
    DetachedSign(src, dst string) error ; ClearSign(src, dst string) error
}
type Verifier interface {
    InitKeyring(verbose bool) error ; AddKeyring(keyring string)
    VerifyDetachedSignature(sig, cleartext io.Reader, showKeyTip bool) error
    IsClearSigned(io.Reader) (bool, error)
    VerifyClearsigned(io.Reader, showKeyTip bool) (*KeyInfo, error)
    ExtractClearsigned(io.Reader) (*os.File, error)   // NO verification
}
```

## 10.1 Providers

Selected by `gpg_provider` (config) or `-gpg-provider` (global flag). `AptlyContext.GetSigner()` /
`GetVerifier()` instantiate:

| Provider | Signer | Verifier | Notes |
|----------|--------|----------|-------|
| `gpg` (default) | `GpgSigner` (runs `gpg`) | `GpgVerifier` (runs `gpgv`) | binaries located by `GPGDefaultFinder` (`gpg`, `gpgv`); auto-detects gnupg 1.x vs 2.x |
| `gpg1` | same | same | requires `gpg`/`gpg1` and `gpgv`/`gpgv1` in `$PATH` that are GnuPG **1.x** |
| `gpg2` | same | same | requires `gpg`/`gpg2`, `gpgv`/`gpgv2` = **2.x** |
| `internal` | `GoSigner` | `GoVerifier` | pure Go (`golang.org/x/crypto/openpgp`, plus extended signature-checking helpers in `pgp/openpgp.go` that can validate *several* signers of one detached signature and track missing keys); no external binaries |

`gpg` and `gpgv` versions must match, otherwise `NewGpgVerifier` panics *"gpg and gpgv versions
don't match"*. Test fixtures for this logic are in `pgp/test-bins`.

## 10.2 Verifying remote repositories

### Trust store

* External provider: `gpgv --keyring trustedkeys.gpg …` — i.e. the **`trustedkeys.gpg`** keyring in
  the GnuPG home of the user running aptly (aptly does *not* use the general `pubring`). Extra keyrings
  are added with `-keyring=<file>` (repeatable); if any are given they *replace* the default.
* Internal provider: keyring files are loaded by `loadKeyRing`; names without `/` are relative to
  `$GNUPGHOME` (default `~/.gnupg`). Missing default keyring is tolerated (empty); a missing explicit
  one prints a warning.
* API: `POST /api/gpg/key` (import armored key text or fetch `GpgKeyID`s from a `Keyserver`),
  `GET /api/gpg/keys`, `DELETE /api/gpg/key` — implemented by shelling out to
  `gpg --no-default-keyring --keyring <name> …` (keyring path is sanitised).

Typical bootstrap on Debian/Ubuntu (printed by aptly when the keyring is empty):

```bash
gpg --no-default-keyring --keyring /usr/share/keyrings/debian-archive-keyring.gpg --export \
  | gpg --no-default-keyring --keyring trustedkeys.gpg --import
# or a single key:
gpg --no-default-keyring --keyring trustedkeys.gpg --keyserver keyserver.ubuntu.com --recv-keys 0xKEYID
wget -O - https://vendor.example/Release.key | gpg --no-default-keyring --keyring trustedkeys.gpg --import
```

### What is verified, in which order (`RemoteRepo.Fetch`)

1. `InRelease` (clearsigned) — `VerifyClearsigned`; text extracted with `ExtractClearsigned`.
2. If missing/invalid → `Release` + `Release.gpg` — `VerifyDetachedSignature`.
3. With `-ignore-signatures` (or `gpg_disable_verify`) **no verification** takes place; the payload is
   still parsed (an `InRelease` is stripped of its armour via `ExtractClearsigned`).

Integrity below `Release` relies on hashes, not signatures: every `Packages*` file is checked against
`Release`'s checksums (`ReleaseFiles`), and every `.deb` against the `Packages` stanza checksums
(`DownloadWithChecksum`). Disabling checksum checks (`-ignore-checksums`) weakens the chain.

Failure output (`runGpgv`): prints which keys were **good** vs **missing** (`--status-fd 3`), and
suggests `--recv-keys` for missing key IDs (`KeyInfo{GoodKeys, MissingKeys}`).
`repo add` verifies `.dsc` signatures when present; `repo include` verifies `.changes`
(unless `-ignore-signatures`, `-accept-unsigned`); the signing keys are then used for
uploaders ACL (see [`repo include`](05-workflows.md#52-local-repositories)) decisions.

## 10.3 Signing published repositories

### Which files are signed

| File | Signature |
|------|-----------|
| `dists/<dist>/Release` | detached armored → `Release.gpg`; and clearsigned → `InRelease` |
| installer `…/installer-<arch>/current/images/SHA256SUMS` | detached `SHA256SUMS.gpg` |
| everything else (`Packages`, `Contents`, per-arch `Release`) | *not* individually signed — covered by hashes in the signed top-level `Release` |

Digest algorithm for the external provider is forced to **SHA256** (`--digest-algo SHA256`).

### Choosing the key

Resolved by `getSigner(flags)` (CLI, `cmd/publish.go`) / `getSigner(&signingParams)` (API):

1. `-skip-signing` (or `gpg_disable_sign: true`, or API `Signing.Skip`) ⇒ **no signer**; nothing signed.
2. Keys: `-gpg-key` (repeatable) → else config `gpg_keys` → else *gpg's default key*
   (`GpgSigner` passes each as `-u <key>`; several keys produce several signatures — useful during key
   rotation. API: `Signing.GpgKey = "ID1, ID2"`.)
3. Keyrings: `-keyring`, `-secret-keyring` (GnuPG 1 only for secret keyring).
4. Passphrase: `-passphrase`, `-passphrase-file` (both **insecure** in shell history/process list; prefer a
   file with restrictive mode or gpg-agent), or interactive prompt.
5. `-batch` ⇒ `--no-tty --batch` (never prompt). **The API always uses batch mode** (an interactive prompt
   would block the server); so protected keys need `gpg-agent` with cached passphrase, or a passphrase
   file/param.
6. `signer.Init()` checks that `gpg --list-keys` works and that *some* key exists (when no explicit
   keyring is given).

### GoSigner specifics

`Init()` reads `pubring.gpg` and `secring.gpg` (legacy keyring formats) from `$GNUPGHOME` unless paths
are given; picks the first valid secret key or the one matching `keyRef` (key ID, or substring of a user
ID); decrypts it with the supplied/prompted passphrase (3 attempts interactive). If `SOURCE_DATE_EPOCH`
is set, signatures are created with that timestamp and *deterministic* settings. **Limitation:** modern
GnuPG (2.1+) stores secret keys in `private-keys-v1.d`/keybox, which `internal` cannot read — export the
key into legacy keyring files, or use the `gpg` provider.

### Publishing your public key

aptly never publishes your public key. Export it yourself and host it next to the repository
(skeleton files from `rootDir/skel` are copied *into component directories*, not the prefix root, so
they are not a convenient place for it):

```bash
gpg --armor --export KEYID > /srv/apt/public.key
# client:
curl -fsSL https://repo.example.com/public.key | sudo gpg --dearmor -o /etc/apt/keyrings/example.gpg
# sources.list: deb [signed-by=/etc/apt/keyrings/example.gpg] https://repo.example.com/ stable main
```

`-signed-by=<fingerprints>` on publish adds `Signed-By:` (+ `Valid-Until` 100 years ahead) to the top-level
`Release`, letting clients pin acceptable keys for future updates.

## 10.4 Security notes

* **Passphrases on the command line** are visible in `ps` and shell history; the API
  `Signing.Passphrase` field is stored nowhere but travels in the request body — use TLS.
* The private key must be available on the machine running aptly; consider a dedicated *repository*
  subkey with expiry, and rotate using multiple `-gpg-key`s.
* Verification via `trustedkeys.gpg` is only as strong as its contents: import vendor keys from their
  official keyrings, not via TOFU keyserver fetches, when possible.
* Never use `-ignore-signatures`/`-ignore-checksums` on untrusted networks.
