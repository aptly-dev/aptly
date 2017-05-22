package api

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

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
	args := []string{"--no-default-keyring"}
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
		tempdir, err = ioutil.TempDir(os.TempDir(), "aptly")
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		defer os.RemoveAll(tempdir)

		keypath := filepath.Join(tempdir, "key")
		keyfile, e := os.Create(keypath)
		if e != nil {
			c.AbortWithError(400, e)
			return
		}
		if _, e = keyfile.WriteString(b.GpgKeyArmor); e != nil {
			c.AbortWithError(400, e)
		}
		args = append(args, "--import", keypath)

	}
	if len(b.GpgKeyID) > 0 {
		args = append(args, "--recv", b.GpgKeyID)
	}

	// it might happened that we have a situation with an erroneous
	// gpg command (e.g. when GpgKeyID and GpgKeyArmor is set).
	// there is no error handling for such as gpg will do this for us
	cmd := exec.Command("gpg", args...)
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		c.AbortWithError(400, err)
		return
	}

	c.JSON(200, gin.H{})
}
