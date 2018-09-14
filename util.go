package main

import (
	"fmt"
	"os"
)

func logError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

func createPathIfNotExist(path string, perm os.FileMode) error {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, perm); err != nil {
			return logError(err.Error())
		}
	}
	return nil
}
