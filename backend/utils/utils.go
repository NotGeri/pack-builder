package utils

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

type Simple struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Tracker struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

type H map[string]interface{}

// SendJSON sends application/json encoded data back to a response writer
func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GetRegexGroup returns the value of a single regex
// group from a given pattern and data to match against
func GetRegexGroup(regex *regexp.Regexp, key, data string) string {
	groups := GetRegexGroups(regex, data)
	return groups[key]
}

// GetRegexGroups returns a map of all regex groups and their values
func GetRegexGroups(regex *regexp.Regexp, data string) (paramsMap map[string]string) {

	match := regex.FindStringSubmatch(data)

	paramsMap = make(map[string]string)
	for i, name := range regex.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}

	return paramsMap
}

type ZipInfo struct {
	Path string
	Size int64
}

func ZipFolder(zipPath, folderPath string) (info ZipInfo, err error) {
	// Create a new zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return
	}

	zipWriter := zip.NewWriter(zipFile)

	defer func() {
		zipFile.Close()
		zipWriter.Close()
	}()

	// Get the absolute path of the folder
	absFolderPath, err := filepath.Abs(folderPath)
	if err != nil {
		return
	}

	// Walk through the folderPath
	err = filepath.Walk(absFolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path
		relPath, err := filepath.Rel(absFolderPath, path)
		if err != nil {
			return err
		}

		// Skip the root folder itself
		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			return err
		}

		return nil
	})

	if err != nil {
		return
	}

	// Close the zip writer to ensure all data is written
	if err = zipWriter.Close(); err != nil {
		return
	}

	// Get the size of the created zip file
	fileInfo, err := zipFile.Stat()
	if err != nil {
		return
	}

	info = ZipInfo{
		Path: zipFile.Name(),
		Size: fileInfo.Size(),
	}

	return
}
