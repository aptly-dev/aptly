package main

import (
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
)

func main() {
	downloader := utils.NewDownloader(2)
	defer downloader.Shutdown()

	database, _ := database.OpenDB("/tmp/aptlydb")
	defer database.Close()

	repo, _ := debian.NewRemoteRepo("http://mirror.yandex.ru/debian/", "squeeze", []string{}, []string{})
	err := repo.Fetch(downloader)
	fmt.Printf("Fetch(), err = %#v", err)

	err = repo.Download(downloader, database)
	fmt.Printf("Download(), err = %#v", err)
}
