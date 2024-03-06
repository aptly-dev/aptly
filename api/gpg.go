package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/gin-gonic/gin"
)

// POST /api/gpg
func apiGPGAddKey(c *gin.Context) {
	var b struct {
		Keyserver   string
		GpgKeyID    string
		GpgKeyArmor string
		Keyring     string
	}

	if c.Bind(&b) != nil {
		return
	}

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
