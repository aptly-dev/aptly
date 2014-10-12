package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func verifyDir(c *gin.Context) bool {
	dir := c.Params.ByName("dir")
	dir = filepath.Clean(dir)
	for _, part := range strings.Split(dir, string(filepath.Separator)) {
		if part == ".." || part == "." {
			c.Fail(400, fmt.Errorf("wrong dir"))
			return false
		}
	}

	return true
}

// GET /files
func apiFilesListDirs(c *gin.Context) {
	list := []string{}

	err := filepath.Walk(context.UploadPath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == context.UploadPath() {
			return nil
		}

		if info.IsDir() {
			list = append(list, filepath.Base(path))
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		c.Fail(400, err)
		return
	}

	c.JSON(200, list)
}

// POST /files/:dir/
func apiFilesUpload(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	path := filepath.Join(context.UploadPath(), c.Params.ByName("dir"))
	err := os.MkdirAll(path, 0777)

	if err != nil {
		c.Fail(500, err)
		return
	}

	err = c.Request.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		c.Fail(400, err)
		return
	}

	stored := []string{}

	for _, files := range c.Request.MultipartForm.File {
		for _, file := range files {
			src, err := file.Open()
			if err != nil {
				c.Fail(500, err)
				return
			}
			defer src.Close()

			destPath := filepath.Join(path, filepath.Base(file.Filename))
			dst, err := os.Create(destPath)
			if err != nil {
				c.Fail(500, err)
				return
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			if err != nil {
				c.Fail(500, err)
				return
			}

			stored = append(stored, filepath.Join(c.Params.ByName("dir"), filepath.Base(file.Filename)))
		}
	}

	c.JSON(200, stored)

}

// GET /files/:dir
func apiFilesListFiles(c *gin.Context) {

}

// DELETE /files/:dir
func apiFilesDeleteDir(c *gin.Context) {

}

// DELETE /files/:dir/:name
func apiFilesDeleteFile(c *gin.Context) {

}
