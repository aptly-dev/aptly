package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
)

type gpgAddKeyParams struct {
	// Keyserver, when downloading GpgKeyIDs
	Keyserver string `json:"Keyserver"       example:"hkp://keyserver.ubuntu.com:80"`
	// GpgKeyIDs to download from Keyserver, comma separated list
	GpgKeyID string `json:"GpgKeyID"        example:"EF0F382A1A7B6500,8B48AD6246925553"`
	// Armored gpg public ket, instead of downloading from keyserver
	GpgKeyArmor string `json:"GpgKeyArmor"     example:""`
	// Keyring for adding the keys (default: trustedkeys.gpg)
	Keyring string `json:"Keyring"         example:"trustedkeys.gpg"`
}

// @Summary Add GPG Keys
// @Description **Adds GPG keys to aptly keyring**
// @Description
// @Description Add GPG public keys for veryfing remote repositories for mirroring.
// @Tags Mirrors
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Not Found"
// @Router /api/gpg [post]
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
		defer os.RemoveAll(tempdir)

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
