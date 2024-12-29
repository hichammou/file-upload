package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	allowedMIMETypes map[string]bool = map[string]bool{
		"image/png":       true,
		"image/jpeg":      true,
		"application/pdf": true,
	}
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /upload", uploadFile)

	srv := &http.Server{
		Addr:    ":9001",
		Handler: mux,
	}

	err := srv.ListenAndServe()
	log.Fatal(err)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 10 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to retrieve file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file MIME type
	fileHeader := make([]byte, 512) // Read first 512 bytes for MIME detection
	if _, err = file.Read(fileHeader); err != nil {
		http.Error(w, "Could not read file", http.StatusInternalServerError)
		return
	}
	file.Seek(0, io.SeekStart)

	detectMimeType := http.DetectContentType(fileHeader)
	if !allowedMIMETypes[detectMimeType] {
		http.Error(w, fmt.Sprintf("%s is not allowed", detectMimeType), http.StatusBadRequest)
		return
	}

	// Sanitize filename to prevent file traversal attacks
	safeFilename := filepath.Base(header.Filename)

	destFolder := "images"
	if strings.Contains(detectMimeType, "pdf") {
		destFolder = "pdfs"
	}

	// Create uploads directory if not exists
	uploadsDir := filepath.Join("uploads", destFolder)
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadsDir, os.ModePerm)
		if err != nil {
			if !errors.Is(err, os.ErrExist) {
				log.Printf("Error while creating uplods directory %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
	}

	// Create destination file
	dstPath := filepath.Join(uploadsDir, safeFilename)
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("Error creating destination path %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer dst.Close()

	// Write file to destination
	if _, err = io.Copy(dst, file); err != nil {
		log.Printf("Error writing file %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("File %s uploaded successfully", safeFilename)))
}
