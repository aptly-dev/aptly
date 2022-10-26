package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func verifyPath(path string) bool {
	path = filepath.Clean(path)
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".." || part == "." {
			return false
		}
	}

	return true

}

func verifyDir(c *gin.Context) bool {
	if !verifyPath(c.Params.ByName("dir")) {
		AbortWithJSONError(c, 400, fmt.Errorf("wrong dir"))
		return false
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
		AbortWithJSONError(c, 400, err)
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
		AbortWithJSONError(c, 500, err)
		return
	}

	err = c.Request.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		AbortWithJSONError(c, 400, err)
		return
	}

	stored := []string{}

	for _, files := range c.Request.MultipartForm.File {
		for _, file := range files {
			src, err := file.Open()
			if err != nil {
				AbortWithJSONError(c, 500, err)
				return
			}
			defer src.Close()

			destPath := filepath.Join(path, filepath.Base(file.Filename))
			dst, err := os.Create(destPath)
			if err != nil {
				AbortWithJSONError(c, 500, err)
				return
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			if err != nil {
				AbortWithJSONError(c, 500, err)
				return
			}

			stored = append(stored, filepath.Join(c.Params.ByName("dir"), filepath.Base(file.Filename)))
		}
	}

	c.JSON(200, stored)

}

// GET /files/:dir
func apiFilesListFiles(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	list := []string{}
	root := filepath.Join(context.UploadPath(), c.Params.ByName("dir"))

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		list = append(list, filepath.Base(path))

		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			AbortWithJSONError(c, 404, err)
		} else {
			AbortWithJSONError(c, 500, err)
		}
		return
	}

	c.JSON(200, list)
}

// DELETE /files/:dir
func apiFilesDeleteDir(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	err := os.RemoveAll(filepath.Join(context.UploadPath(), c.Params.ByName("dir")))
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	c.JSON(200, gin.H{})
}

// DELETE /files/:dir/:name
func apiFilesDeleteFile(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	if !verifyPath(c.Params.ByName("name")) {
		AbortWithJSONError(c, 400, fmt.Errorf("wrong file"))
		return
	}

	err := os.Remove(filepath.Join(context.UploadPath(), c.Params.ByName("dir"), c.Params.ByName("name")))
	if err != nil {
		if err1, ok := err.(*os.PathError); !ok || !os.IsNotExist(err1.Err) {
			AbortWithJSONError(c, 500, err)
			return
		}
	}

	c.JSON(200, gin.H{})
}
