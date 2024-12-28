package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	// Sanitize filename to prevent file traversal attacks
	safeFilename := filepath.Base(header.Filename)

	// Create uploads directory if not exists
	uploadsDir := "./uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err = os.Mkdir("uploads", os.ModePerm)
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
