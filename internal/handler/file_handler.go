package handler

import (
	"encoding/json"
	"fileStation/internal/service"
	"fileStation/pkg/logger"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
)

// FileHandler отвечает за обработку запросов, связанных с файлами.
type FileHandler struct {
	fileService *service.FileService
	templates   *template.Template
	authService *service.AuthService
    version     string
}

func NewFileHandler(fileService *service.FileService, templates *template.Template, authService *service.AuthService, version string) *FileHandler {
    return &FileHandler{
        fileService: fileService,
        templates:   templates,
        authService: authService,
        version:     version,
    }
}


// ServeFiles обрабатывает запросы для отображения файлов и папок.
func (h *FileHandler) ServeFiles(w http.ResponseWriter, r *http.Request) {
    reqPath := r.URL.Path
    fullPath := h.fileService.GetFullPath(reqPath)

    // Check if the path exists
    info, err := os.Stat(fullPath)
    if err != nil {
        http.NotFound(w, r)
        return
    }

    // Check authorization
    isLoggedIn := h.isLoggedIn(r)
    username := ""
    if isLoggedIn {
        // Get username from session token
        cookie, err := r.Cookie("session_token")
        if err == nil {
            username, _ = h.authService.GetSessionUsername(cookie.Value)
        }
    }

    if info.IsDir() {
        if !strings.HasSuffix(reqPath, "/") {
            http.Redirect(w, r, reqPath+"/", http.StatusMovedPermanently)
            return
        }

        // List files in the directory
        files, err := h.fileService.ListDirectory(fullPath)
        if err != nil {
            http.Error(w, "Error reading directory", http.StatusInternalServerError)
            return
        }

        // Get modification times
        modTimes, err := h.fileService.GetModificationTimes(fullPath)
        if err != nil {
            http.Error(w, "Error reading directory", http.StatusInternalServerError)
            return
        }

        // Process README.md
        readmeHTML := h.getReadmeHTML(fullPath)

        // Calculate ParentDir
        parentDir := "/"
        if reqPath != "/" {
            cleanedPath := strings.TrimSuffix(reqPath, "/")
            parentDir = filepath.Dir(cleanedPath)
            if parentDir == "." || parentDir == "" {
                parentDir = "/"
            }
            if !strings.HasSuffix(parentDir, "/") {
                parentDir += "/"
            }
        }

        pageTitle := "fileStation - " + reqPath

        // Data for the template
        data := struct {
            Title      string
            Path       string
            ParentDir  string
            FullPath   string
            Files      []os.DirEntry
            ModTimes   map[string]time.Time
            IsLoggedIn bool
            Username   string
            ReadmeHTML template.HTML
            Version    string
        }{
            Title:      pageTitle,
            Path:       reqPath,
            ParentDir:  parentDir,
            FullPath:   fullPath,
            Files:      files,
            ModTimes:   modTimes,
            IsLoggedIn: isLoggedIn,
            Username:   username,
            ReadmeHTML: readmeHTML,
            Version:    h.version,
        }

        h.renderTemplate(w, "index.html", data)
    } else {
        // Serve the file
        http.ServeFile(w, r, fullPath)
    }
}


// DownloadHandler обрабатывает запросы на скачивание файлов.
func (h *FileHandler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	items := r.Form["items"]
	if len(items) == 0 {
		http.Error(w, "No files selected for download", http.StatusBadRequest)
		return
	}

	// Обработка одиночного файла
	if len(items) == 1 {
		fullPath := h.fileService.GetFullPath(items[0])
		http.ServeFile(w, r, fullPath)
		return
	}

	// Архивирование и отправка нескольких файлов
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"files.zip\"")
	if err := h.fileService.CreateZipArchive(w, items); err != nil {
		http.Error(w, "Error creating ZIP archive", http.StatusInternalServerError)
	}
}

// Helper: Проверка, авторизован ли пользователь
func (h *FileHandler) isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	return err == nil && cookie.Value != ""
}

// Helper: Рендеринг HTML-шаблонов
func (h *FileHandler) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
    err := h.templates.ExecuteTemplate(w, "base.html", data)
    if err != nil {
        logger.Errorf("Error rendering template %s: %v", tmpl, err)
        http.Error(w, "Error rendering template", http.StatusInternalServerError)
    }
}

// Helper: Чтение и преобразование README.md
func (h *FileHandler) getReadmeHTML(fullPath string) template.HTML {
	readmePath := filepath.Join(fullPath, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return ""
	}

	var buf strings.Builder
	err = goldmark.Convert(content, &buf)
	if err != nil {
		return ""
	}
	return template.HTML(buf.String())
}


// DirTreeHandler возвращает данные дерева директорий в формате JSON.
func (h *FileHandler) DirTreeHandler(w http.ResponseWriter, r *http.Request) {
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

// ListFoldersHandler возвращает список папок в указанной директории.
func (h *FileHandler) ListFoldersHandler(w http.ResponseWriter, r *http.Request) {
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

// UploadHandler обрабатывает запросы на загрузку файлов.
func (h *FileHandler) UploadHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    err := r.ParseMultipartForm(100 << 20) // Ограничение: 100 MB
    if err != nil {
        http.Error(w, "Error parsing form", http.StatusBadRequest)
        return
    }

    reqPath := r.FormValue("currentPath")
    version := r.FormValue("fileVersion") // Получение версии из формы

    fullDestPath := h.fileService.GetFullPath(reqPath)

    files := r.MultipartForm.File["uploadFiles"]
    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            http.Error(w, "Error reading file", http.StatusBadRequest)
            return
        }
        defer file.Close()

        dstPath := filepath.Join(fullDestPath, fileHeader.Filename)
        err = h.fileService.SaveFile(dstPath, file)
        if err != nil {
            http.Error(w, "Error saving file", http.StatusInternalServerError)
            return
        }

        // Добавление метаданных (версия)
        err = h.fileService.AddMetadata(dstPath, map[string]string{
            "Version": version, // Пустая строка по умолчанию, если версия не указана
        })
        if err != nil {
            http.Error(w, "Error saving metadata", http.StatusInternalServerError)
            return
        }
    }

    http.Redirect(w, r, reqPath, http.StatusSeeOther)
}

// DeleteHandler обрабатывает запросы на удаление файлов и папок.
func (h *FileHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	items := r.Form["items"]
	if len(items) == 0 {
		http.Error(w, "No items selected for deletion", http.StatusBadRequest)
		return
	}

	for _, item := range items {
		fullPath := h.fileService.GetFullPath(item)
		err := h.fileService.Delete(fullPath)
		if err != nil {
			http.Error(w, "Error deleting item", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// CreateFolderHandler обрабатывает запросы на создание папок.
func (h *FileHandler) CreateFolderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqPath := r.FormValue("currentPath")
	folderName := r.FormValue("folderName")
	if folderName == "" {
		http.Error(w, "Folder name is required", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(h.fileService.GetFullPath(reqPath), folderName)
	err := h.fileService.CreateFolder(fullPath)
	if err != nil {
		http.Error(w, "Error creating folder", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, reqPath, http.StatusSeeOther)
}

// RenameHandler обрабатывает запросы на переименование файлов и папок.
func (h *FileHandler) RenameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	oldPath := r.FormValue("oldPath")
	newName := r.FormValue("newName")
	if oldPath == "" || newName == "" {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	fullOldPath := h.fileService.GetFullPath(oldPath)
	fullNewPath := filepath.Join(filepath.Dir(fullOldPath), newName)

	err := h.fileService.Rename(fullOldPath, fullNewPath)
	if err != nil {
		http.Error(w, "Error renaming item", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, filepath.Dir(oldPath), http.StatusSeeOther)
}

// MoveHandler обрабатывает запросы на перемещение файлов и папок.
func (h *FileHandler) MoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemPathsJSON := r.FormValue("itemPaths")
	destinationPath := r.FormValue("destinationPath")

	var itemPaths []string
	err := json.Unmarshal([]byte(itemPathsJSON), &itemPaths)
	if err != nil {
		http.Error(w, "Invalid item paths", http.StatusBadRequest)
		return
	}

	for _, itemPath := range itemPaths {
		fullItemPath := h.fileService.GetFullPath(itemPath)
		fullDestinationPath := filepath.Join(h.fileService.GetFullPath(destinationPath), filepath.Base(itemPath))

		err := h.fileService.Move(fullItemPath, fullDestinationPath)
		if err != nil {
			http.Error(w, "Error moving item", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, destinationPath, http.StatusSeeOther)
}

