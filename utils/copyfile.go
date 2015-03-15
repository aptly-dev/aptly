package utils

import (
	"io"
	"os"
)

// CopyFile copeis file from src to dst, not preserving attributes
func CopyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(df, sf)
	if err != nil {
		df.Close()
		return err
	}

	return df.Close()
}
