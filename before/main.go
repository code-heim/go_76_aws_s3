package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	uploadDir    = "uploads/"
	templateFile = "templates/index.html"
)

func main() {
	// Ensure the upload directory exists
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		panic(err)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "Unable to read images", http.StatusInternalServerError)
		return
	}

	var images []string
	for _, file := range files {
		images = append(images, "/uploads/"+file.Name())
	}

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, images)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the file
	savePath := filepath.Join(uploadDir, header.Filename)
	outFile, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
