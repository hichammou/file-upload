package main

import (
	"errors"
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
	err := r.ParseMultipartForm(20 << 10)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer file.Close()

	log.Printf("name: %v, size: %v, ", handler.Filename, handler.Size)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Print(err, "io.ReadAll")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = os.Mkdir("uploads", os.ModePerm)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Print(err, " os.Mkdir")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	uploadsDir := "./uploads"

	dst, err := os.Create(filepath.Join(uploadsDir, handler.Filename))
	if err != nil {
		log.Print(err, " os.Create")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer dst.Close()
	_, err = dst.Write(fileBytes)
	if err != nil {
		log.Print(err, "dst.Write")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}
