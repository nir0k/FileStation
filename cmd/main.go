package main

import (
	"fileStation/internal/handler"
	"fileStation/internal/service"
	"fileStation/pkg/logger"
	"html/template"
	"net/http"
)

func main() {
	// Initialize services
	authService := service.NewAuthService()
	fileService := service.NewFileService("/path/to/base/dir", authService)

	// Parse templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Initialize handlers
	fileHandler := handler.NewFileHandler(fileService, templates, authService, "1.0.0")

	// Define routes
	http.HandleFunc("/", fileHandler.ServeFiles)
	http.HandleFunc("/download", fileHandler.DownloadHandler)
	http.HandleFunc("/upload", fileHandler.UploadHandler)
	http.HandleFunc("/delete", fileHandler.DeleteHandler)
	http.HandleFunc("/create-folder", fileHandler.CreateFolderHandler)
	http.HandleFunc("/rename", fileHandler.RenameHandler)
	http.HandleFunc("/move", fileHandler.MoveHandler)
	http.HandleFunc("/file-metadata", fileHandler.FileMetadataHandler)
	http.HandleFunc("/recalculate-hashes", fileHandler.RecalculateHashesHandler)
	http.HandleFunc("/save-metadata", fileHandler.SaveMetadataHandler)

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server
	logger.Info("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatalf("Could not start server: %v", err)
	}
}