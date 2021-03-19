package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	osName string
)

const homeDrive = "Home"

func init() {
	if runtime.GOOS == "darwin" {
		osName = "darwin"
	} else if runtime.GOOS == "windows" {
		osName = "windows"
	} else if runtime.GOOS == "linux" {
		osName = "linux"
	}
}

// TODO: Fill this for windows
func ListDrives() ([]string, error) {
	if osName == "windows" {
		// TODO: Create list drive detection in windows
		return nil, nil
	}
	return []string{homeDrive}, nil
}

// TODO: Fill this for windows
func GetDriveLabel(drive string) (string, error) {
	if osName == "windows" {

		return "", nil
	}
	return "Home root", nil
}

// Returns all files inside the specified directory
func GetFilesIn(path string) ([]string, error) {
	files := []string{}
	f, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range f {
		if file.IsDir() {
			continue
		}
		files = append(files, file.Name())
	}

	return files, nil
}

// Returns all directories inside the specified path
func GetDirectoriesIn(path string) ([]string, error) {
	dirs := []string{}
	f, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range f {
		if !file.IsDir() {
			continue
		}
		dirs = append(dirs, filepath.Join(path, file.Name()))
	}

	return dirs, nil
}

func NormalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\\\", "/")
	path = strings.ReplaceAll(path, "//", "/")
	if osName != "windows" {
		return homeDrive + ":" + path
	}
	return path
}

func DenormalizePath(path string) string {
	if osName != "windows" {
		if strings.HasPrefix(path, homeDrive+":") {
			return strings.ReplaceAll(path, homeDrive+":", "")
		}
	}
	return strings.ReplaceAll(path, "/", "\\\\")
}

// Deletes specified path and all its contents
func DeletePath(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	return nil
}
