package api

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
)

type gpgListKeysParams struct {
	// Keyring to list keys from (default: trustedkeys.gpg)
	Keyring string `json:"Keyring"         example:"trustedkeys.gpg"`
}

type gpgKeyInfo struct {
	// 16-character key ID (short form)
	KeyID string `json:"KeyID"         example:"8B48AD6246925553"`
	// Full fingerprint
	Fingerprint string `json:"Fingerprint"   example:"D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0"`
	// Key validity (u=unknown, f=fulltrust, m=marginal, n=never)
	Validity string `json:"Validity"      example:"u"`
	// User ID(s) associated with this key
	UserIDs []string `json:"UserIDs"       example:"John Doe <john@example.com>"`
	// Creation date (Unix timestamp format from gpg)
	CreatedAt string `json:"CreatedAt"     example:"2023-01-15"`
}

type gpgKeyListResponse struct {
	Keys []gpgKeyInfo `json:"Keys"`
}

type gpgAddKeyParams struct {
	// Keyring for adding the keys (default: trustedkeys.gpg)
	Keyring string `json:"Keyring"         example:"trustedkeys.gpg"`

	// Add ASCII armored gpg public key, do not download from keyserver
	GpgKeyArmor string `json:"GpgKeyArmor"     example:""`

	// Keyserver to download keys provided in `GpgKeyID`
	Keyserver string `json:"Keyserver"       example:"hkp://keyserver.ubuntu.com:80"`
	// Keys do download from `Keyserver`, separated by space
	GpgKeyID string `json:"GpgKeyID"        example:"EF0F382A1A7B6500 8B48AD6246925553"`
}

type gpgDeleteKeyParams struct {
	// Keyring to delete keys from (default: trustedkeys.gpg)
	Keyring string `json:"Keyring"         example:"trustedkeys.gpg"`

	// Key ID or fingerprint to delete
	GpgKeyID string `json:"GpgKeyID"        example:"8B48AD6246925553"`
}

// @Summary Add GPG Keys
// @Description **Adds GPG keys to aptly keyring**
// @Description
// @Description Add GPG public keys for veryfing remote repositories for mirroring.
// @Description
// @Description Keys can be added in two ways:
// @Description * By providing the ASCII armord key in `GpgKeyArmor` (leave Keyserver and GpgKeyID empty)
// @Description * By providing a `Keyserver` and one or more key IDs in `GpgKeyID`, separated by space (leave GpgKeyArmor empty)
// @Description
// @Tags Mirrors
// @Consume  json
// @Param request body gpgAddKeyParams true "Parameters"
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 400 {object} Error "Bad Request"
// @Router /api/gpg/key [post]
func apiGPGAddKey(c *gin.Context) {
	b := gpgAddKeyParams{}
	if c.Bind(&b) != nil {
		return
	}
	b.Keyserver = utils.SanitizePath(b.Keyserver)
	b.GpgKeyID = utils.SanitizePath(b.GpgKeyID)
	b.GpgKeyArmor = utils.SanitizePath(b.GpgKeyArmor)
	// b.Keyring can be an absolute path

	var err error
	args := []string{"--no-default-keyring", "--allow-non-selfsigned-uid"}
	keyring := "trustedkeys.gpg"
	if len(b.Keyring) > 0 {
		keyring = b.Keyring
	}
	args = append(args, "--keyring", keyring)
	if len(b.Keyserver) > 0 {
		args = append(args, "--keyserver", b.Keyserver)
	}
	if len(b.GpgKeyArmor) > 0 {
		var tempdir string
		tempdir, err = os.MkdirTemp(os.TempDir(), "aptly")
		if err != nil {
			AbortWithJSONError(c, 400, err)
			return
		}
		defer func() { _ = os.RemoveAll(tempdir) }()

		keypath := filepath.Join(tempdir, "key")
		keyfile, e := os.Create(keypath)
		if e != nil {
			AbortWithJSONError(c, 400, e)
			return
		}
		if _, e = keyfile.WriteString(b.GpgKeyArmor); e != nil {
			AbortWithJSONError(c, 400, e)
		}
		args = append(args, "--import", keypath)

	}
	if len(b.GpgKeyID) > 0 {
		keys := strings.Fields(b.GpgKeyID)
		args = append(args, "--recv-keys")
		args = append(args, keys...)
	}

	finder := pgp.GPGDefaultFinder()
	gpg, _, err := finder.FindGPG()
	if err != nil {
		AbortWithJSONError(c, 400, err)
		return
	}

	// it might happened that we have a situation with an erroneous
	// gpg command (e.g. when GpgKeyID and GpgKeyArmor is set).
	// there is no error handling for such as gpg will do this for us
	cmd := exec.Command(gpg, args...)
	fmt.Printf("running %s %s\n", gpg, strings.Join(args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(400, string(out))
		return
	}

	c.JSON(200, string(out))
}

// @Summary List GPG Keys
// @Description **Lists all GPG keys in aptly keyring**
// @Description
// @Description Returns all public keys currently installed in the aptly GPG keyring.
// @Description
// @Tags Mirrors
// @Param keyring query string false "Keyring file to list keys from (default: trustedkeys.gpg)" example(trustedkeys.gpg)
// @Produce json
// @Success 200 {object} gpgKeyListResponse "OK"
// @Failure 400 {object} Error "Bad Request"
// @Router /api/gpg/keys [get]
func apiGPGListKeys(c *gin.Context) {
	keyring := c.DefaultQuery("keyring", "trustedkeys.gpg")
	keyring = utils.SanitizePath(keyring)

	finder := pgp.GPGDefaultFinder()
	gpg, _, err := finder.FindGPG()
	if err != nil {
		AbortWithJSONError(c, 400, err)
		return
	}

	args := []string{
		"--no-default-keyring",
		"--with-colons",
		"--keyring", keyring,
		"--list-keys",
	}

	cmd := exec.Command(gpg, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("failed to list keys: %s", string(out)))
		return
	}

	keys := parseGPGOutput(string(out))
	c.JSON(200, gpgKeyListResponse{Keys: keys})
}

// @Summary Delete GPG Key
// @Description **Deletes a GPG key from aptly keyring**
// @Description
// @Description Removes a public key from the aptly GPG keyring. This is useful for removing
// @Description compromised keys or cleaning up obsolete keys.
// @Description
// @Tags Mirrors
// @Consume  json
// @Param request body gpgDeleteKeyParams true "Parameters"
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 400 {object} Error "Bad Request"
// @Router /api/gpg/key [delete]
func apiGPGDeleteKey(c *gin.Context) {
	b := gpgDeleteKeyParams{}
	if c.Bind(&b) != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("invalid request body"))
		return
	}

	if len(strings.TrimSpace(b.GpgKeyID)) == 0 {
		AbortWithJSONError(c, 400, fmt.Errorf("GpgKeyID is required"))
		return
	}

	b.GpgKeyID = utils.SanitizePath(b.GpgKeyID)
	// b.Keyring can be an absolute path

	finder := pgp.GPGDefaultFinder()
	gpg, _, err := finder.FindGPG()
	if err != nil {
		AbortWithJSONError(c, 400, err)
		return
	}

	args := []string{
		"--no-default-keyring",
		"--allow-non-selfsigned-uid",
	}

	keyring := "trustedkeys.gpg"
	if len(b.Keyring) > 0 {
		keyring = b.Keyring
	}

	args = append(args, "--keyring", keyring)
	args = append(args, "--delete-keys", b.GpgKeyID)

	cmd := exec.Command(gpg, args...)
	fmt.Printf("running %s %s\n", gpg, strings.Join(args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("failed to delete key: %s", string(out)))
		return
	}

	c.JSON(200, string(out))
}

// parseGPGOutput parses the output of `gpg --with-colons --list-keys`
// and returns a structured list of keys
func parseGPGOutput(output string) []gpgKeyInfo {
	var keys []gpgKeyInfo
	var currentKey *gpgKeyInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 10 {
			continue
		}

		recordType := parts[0]

		// pub: public key record
		if recordType == "pub" {
			// Save previous key if it exists
			if currentKey != nil && currentKey.KeyID != "" {
				keys = append(keys, *currentKey)
			}

			// Create new key entry
			// Format: pub:trust:length:algo:keyid:created:expires:uidhash:...
			keyID := parts[4]
			if len(keyID) >= 16 {
				keyID = keyID[len(keyID)-16:] // Last 16 chars = short key ID
			}
			validity := parts[1]
			createdAt := parts[5]

			currentKey = &gpgKeyInfo{
				KeyID:      keyID,
				Validity:   validity,
				CreatedAt:  createdAt,
				UserIDs:    []string{},
				Fingerprint: "",
			}
		}

		// uid: user ID record
		if recordType == "uid" && currentKey != nil {
			// Format: uid:trust:created:expires:keyid:uidhash:uidtype:validity:userID:...
			if len(parts) >= 10 {
				userID := parts[9]
				if userID != "" {
					currentKey.UserIDs = append(currentKey.UserIDs, userID)
				}
			}
		}

		// fpr: fingerprint record
		if recordType == "fpr" && currentKey != nil {
			// Format: fpr:::::::::fingerprint:
			if len(parts) >= 10 {
				fingerprint := parts[9]
				currentKey.Fingerprint = fingerprint
			}
		}
	}

	// Don't forget the last key
	if currentKey != nil && currentKey.KeyID != "" {
		keys = append(keys, *currentKey)
	}

	return keys
}
