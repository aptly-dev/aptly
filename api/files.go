package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
	"github.com/saracen/walker"
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

// @Summary List Directories
// @Description **Get list of upload directories**
// @Description
// @Description **Example:**
// @Description  ```
// @Description  $ curl http://localhost:8080/api/files
// @Description  ["aptly-0.9"]
// @Description  ```
// @Tags Files
// @Produce json
// @Success 200 {array} string "List of files"
// @Router /api/files [get]
func apiFilesListDirs(c *gin.Context) {
	list := []string{}
	listLock := &sync.Mutex{}

	err := walker.Walk(context.UploadPath(), func(path string, info os.FileInfo) error {
		if path == context.UploadPath() {
			return nil
		}

		if info.IsDir() {
			listLock.Lock()
			defer listLock.Unlock()
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

// @Summary Upload Files
// @Description **Upload files to a directory**
// @Description
// @Description - one or more files can be uploaded
// @Description - existing uploaded are overwritten
// @Description
// @Description **Example:**
// @Description  ```
// @Description $ curl -X POST -F file=@aptly_0.9~dev+217+ge5d646c_i386.deb http://localhost:8080/api/files/aptly-0.9
// @Description ["aptly-0.9/aptly_0.9~dev+217+ge5d646c_i386.deb"]
// @Description  ```
// @Tags Files
// @Accept multipart/form-data
// @Param dir path string true "Directory to upload files to. Created if does not exist"
// @Param files formData file true "Files to upload"
// @Produce json
// @Success 200 {array} string "list of uploaded files"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Not Found"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/files/{dir} [post]
func apiFilesUpload(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	path := filepath.Join(context.UploadPath(), utils.SanitizePath(c.Params.ByName("dir")))
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

	apiFilesUploadedCounter.WithLabelValues(c.Params.ByName("dir")).Inc()
	c.JSON(200, stored)
}

// @Summary Upload One File
// @Description **Upload one file to a directory**
// @Description
// @Description - file is uploaded
// @Description - existing uploaded are overwritten
// @Description
// @Description **Example:**
// @Description  ```
// @Description $ dput aptly aptly_0.9~dev+217+ge5d646c_i386.changes
// @Description  ```
// @Tags Files
// @Param dir path string true "Directory to upload files to. Created if does not exist"
// @Param file path string true "File to upload"
// @Produce json
// @Success 200 {array} string "Name of uploaded file"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Not Found"
// @Failure 500 {object} Error "Internal Server Error"
func apiFilesUploadOne(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	path := filepath.Join(context.UploadPath(), utils.SanitizePath(c.Params.ByName("dir")))
	err := os.MkdirAll(path, 0777)

	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}
	stored := []string{}

	destPath := filepath.Join(path, c.Params.ByName("file"))
	dst, err := os.Create(destPath)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}
	defer dst.Close()

	buf := make([]byte, 1024)
	for {
		n, err := c.Request.Body.Read(buf)
		if err != nil && err != io.EOF {
			AbortWithJSONError(c, 400, err)
			return
		}
		if n == 0 {
			break
		}
		if _, err := dst.Write(buf[:n]); err != nil {
			AbortWithJSONError(c, 500, err)
			return
		}
	}

	stored = append(stored, filepath.Join(c.Params.ByName("dir"), c.Params.ByName("file")))

	apiFilesUploadedCounter.WithLabelValues(c.Params.ByName("dir")).Inc()
	c.JSON(200, stored)
}

// @Summary List Files
// @Description **Show uploaded files in upload directory**
// @Description
// @Description **Example:**
// @Description  ```
// @Description $ curl http://localhost:8080/api/files/aptly-0.9
// @Description ["aptly_0.9~dev+217+ge5d646c_i386.deb"]
// @Description  ```
// @Tags Files
// @Produce json
// @Param dir path string true "Directory to list"
// @Success 200 {array} string "Files found in directory"
// @Failure 404 {object} Error "Not Found"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/files/{dir} [get]
func apiFilesListFiles(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	list := []string{}
	listLock := &sync.Mutex{}
	root := filepath.Join(context.UploadPath(), utils.SanitizePath(c.Params.ByName("dir")))

	err := filepath.Walk(root, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		listLock.Lock()
		defer listLock.Unlock()
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

// @Summary Delete Directory
// @Description **Delete upload directory and uploaded files within**
// @Description
// @Description **Example:**
// @Description  ```
// @Description $ curl -X DELETE http://localhost:8080/api/files/aptly-0.9
// @Description {}
// @Description  ```
// @Tags Files
// @Produce json
// @Param dir path string true "Directory"
// @Success 200 {object} string "msg"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/files/{dir} [delete]
func apiFilesDeleteDir(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	err := os.RemoveAll(filepath.Join(context.UploadPath(), utils.SanitizePath(c.Params.ByName("dir"))))
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	c.JSON(200, gin.H{})
}

// @Summary Delete File
// @Description **Delete a uploaded file in upload directory**
// @Description
// @Description **Example:**
// @Description  ```
// @Description $ curl -X DELETE http://localhost:8080/api/files/aptly-0.9/aptly_0.9~dev+217+ge5d646c_i386.deb
// @Description {}
// @Description  ```
// @Tags Files
// @Produce json
// @Param dir path string true "Directory to delete from"
// @Param name path string true "File to delete"
// @Success 200 {object} string "msg"
// @Failure 400 {object} Error "Bad Request"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/files/{dir}/{name} [delete]
func apiFilesDeleteFile(c *gin.Context) {
	if !verifyDir(c) {
		return
	}

	dir := utils.SanitizePath(c.Params.ByName("dir"))
	name := utils.SanitizePath(c.Params.ByName("name"))
	if !verifyPath(name) {
		AbortWithJSONError(c, 400, fmt.Errorf("wrong file"))
		return
	}

	err := os.Remove(filepath.Join(context.UploadPath(), dir, name))
	if err != nil {
		if err1, ok := err.(*os.PathError); !ok || !os.IsNotExist(err1.Err) {
			AbortWithJSONError(c, 500, err)
			return
		}
	}

	c.JSON(200, gin.H{})
}
