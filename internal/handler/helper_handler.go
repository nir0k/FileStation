package handler

import (
	"encoding/json"
	"fileStation/internal/service"
	"net/http"
	"path/filepath"
	"strings"
)

// HelperHandler содержит обработчики вспомогательных операций.
type HelperHandler struct {
	fileService *service.FileService
}

// NewHelperHandler создает новый экземпляр HelperHandler.
func NewHelperHandler(fileService *service.FileService) *HelperHandler {
	return &HelperHandler{
		fileService: fileService,
	}
}

// DirTreeHandler предоставляет данные структуры директорий для jsTree.
func (h *HelperHandler) DirTreeHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	currentPath := r.FormValue("id")
	if currentPath == "" || currentPath == "#" {
		currentPath = "/"
	}
	fullPath := h.fileService.GetFullPath(currentPath)

	// Проверка на выход за пределы базовой директории
	if !strings.HasPrefix(fullPath, h.fileService.GetFullPath("/")) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	entries, err := h.fileService.ListDirectory(fullPath)
	if err != nil {
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}

	var dirs []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			childPath := filepath.Join(currentPath, entry.Name())

			// Проверка наличия дочерних элементов
			hasChildren := false
			childFullPath := filepath.Join(fullPath, entry.Name())
			childEntries, err := h.fileService.ListDirectory(childFullPath)
			if err == nil {
				for _, childEntry := range childEntries {
					if childEntry.IsDir() {
						hasChildren = true
						break
					}
				}
			}

			dirs = append(dirs, map[string]interface{}{
				"id":       childPath,
				"text":     entry.Name(),
				"children": hasChildren,
				"type":     "default",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dirs)
}

// ListFoldersHandler предоставляет список папок в указанном пути.
func (h *HelperHandler) ListFoldersHandler(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		pathParam = "/"
	}

	fullPath := h.fileService.GetFullPath(pathParam)

	// Проверка на выход за пределы базовой директории
	if !strings.HasPrefix(fullPath, h.fileService.GetFullPath("/")) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	entries, err := h.fileService.ListDirectory(fullPath)
	if err != nil {
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}

	var folders []string
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, entry.Name())
		}
	}

	response := map[string]interface{}{
		"folders": folders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
