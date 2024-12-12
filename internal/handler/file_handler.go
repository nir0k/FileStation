package handler

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fileStation/internal/service"
	"fileStation/pkg/logger"
	"fmt"
	"hash/crc32"
	"hash/crc64"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"golang.org/x/crypto/blake2s"
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
		if (!strings.HasSuffix(reqPath, "/")) {
			http.Redirect(w, r, reqPath+"/", http.StatusMovedPermanently)
			return
		}

		// List files in the directory
		entries, err := h.fileService.ListDirectory(fullPath)
		if err != nil {
			http.Error(w, "Error reading directory", http.StatusInternalServerError)
			return
		}

			// Separate entries into folders and files
		var folders, files []os.DirEntry
		for _, entry := range entries {
			if entry.IsDir() {
				folders = append(folders, entry)
			} else {
				files = append(files, entry)
			}
		}

		// Sort folders and files separately
		sort.Slice(folders, func(i, j int) bool {
			return strings.ToLower(folders[i].Name()) < strings.ToLower(folders[j].Name())
		})
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name())
		})

		// Combine folders and files, folders first
		entries = append(folders, files...)

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
			if (!strings.HasSuffix(parentDir, "/")) {
				parentDir += "/"
			}
		}

		pageTitle := "fileStation - " + reqPath

		rdsStatuses := make(map[string]string)
		for _, file := range entries {
			if !file.IsDir() && !strings.HasSuffix(file.Name(), ".md") && !strings.HasSuffix(file.Name(), ".html") && !strings.HasSuffix(file.Name(), ".txt") {
				metaFilePath := filepath.Join(fullPath, "."+file.Name()+".meta")
				metadata, err := h.fileService.ReadMetadata(metaFilePath)
				if err == nil {
					if metadata["RDS CRC32"] == metadata["CRC32"] ||
						metadata["RDS CRC64"] == metadata["CRC64"] ||
						metadata["RDS SHA1"] == metadata["SHA1"] ||
						metadata["RDS SHA256"] == metadata["SHA256"] ||
						metadata["RDS BLAKE2sp"] == metadata["BLAKE2sp"] {
						rdsStatuses[file.Name()] = "match"
					} else if metadata["RDS CRC32"] != "" || metadata["RDS CRC64"] != "" || metadata["RDS SHA1"] != "" || metadata["RDS SHA256"] != "" || metadata["RDS BLAKE2sp"] != "" {
						rdsStatuses[file.Name()] = "mismatch"
					} else {
						rdsStatuses[file.Name()] = "unknown"
					}
				} else if !os.IsNotExist(err) {
					rdsStatuses[file.Name()] = "unknown"
				}
			}
		}

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
			RDSStatuses map[string]string
		}{
			Title:      pageTitle,
			Path:       reqPath,
			ParentDir:  parentDir,
			FullPath:   fullPath,
			Files:      entries,
			ModTimes:   modTimes,
			IsLoggedIn: isLoggedIn,
			Username:   username,
			ReadmeHTML: readmeHTML,
			Version:    h.version,
			RDSStatuses: rdsStatuses,
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

	// Архивирование и отправка файлов и папок
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
		// Check if the error is related to http2 stream closed
		if strings.Contains(err.Error(), "http2: stream closed") {
			logger.Debugf("HTTP2 stream closed error while rendering template %s: %v", tmpl, err)
		} else {
			logger.Errorf("Error rendering template %s: %v", tmpl, err)
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
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
	if (!strings.HasPrefix(fullPath, h.fileService.GetFullPath("/"))) {
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
	if (!strings.HasPrefix(fullPath, h.fileService.GetFullPath("/"))) {
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

	// Get the username from the session
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	username, err := h.authService.GetSessionUsername(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = r.ParseMultipartForm(100 << 20) // Limit: 100 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	reqPath := r.FormValue("currentPath")
	sameVersion := r.FormValue("sameVersion") == "true"

	var version string
	fileVersionMap := make(map[string]string)
	if sameVersion {
		version = r.FormValue("fileVersion")
	} else {
		fileNames := r.Form["fileNames"]
		fileVersions := r.Form["fileVersions"]
		if len(fileNames) != len(fileVersions) {
			http.Error(w, "Mismatch between file names and versions", http.StatusBadRequest)
			return
		}
		for i := 0; i < len(fileNames); i++ {
			fileVersionMap[fileNames[i]] = fileVersions[i]
		}
	}

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

		// Open the destination file
		dstFile, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		// Prepare hash writers
		crc32Hash := crc32.NewIEEE()
		crc64Hash := crc64.New(crc64.MakeTable(crc64.ECMA))
		sha1Hash := sha1.New()
		sha256Hash := sha256.New()
		blake2spHash, err := blake2s.New256(nil)
		if err != nil {
			http.Error(w, "Error creating BLAKE2sp hash", http.StatusInternalServerError)
			return
		}

		// MultiWriter to write to dstFile and compute checksums
		writer := io.MultiWriter(dstFile, crc32Hash, crc64Hash, sha1Hash, sha256Hash, blake2spHash)

		// Copy data from uploaded file to dstFile and compute hashes
		_, err = io.Copy(writer, file)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			return
		}

		dstFile.Close() // Close the destination file

		// Get the checksums
		crc32Checksum := strings.ToUpper(fmt.Sprintf("%x", crc32Hash.Sum32()))
		crc64Checksum := fmt.Sprintf("%x", crc64Hash.Sum(nil))
		sha1Checksum := fmt.Sprintf("%x", sha1Hash.Sum(nil))
		sha256Checksum := fmt.Sprintf("%x", sha256Hash.Sum(nil))
		blake2spChecksum := fmt.Sprintf("%x", blake2spHash.Sum(nil))

		// Determine version
		var versionForFile string
		if sameVersion {
			versionForFile = version
		} else {
			versionForFile = fileVersionMap[fileHeader.Filename]
		}

		// Collect metadata
		metadata := map[string]string{
			"Version":  versionForFile,
			"Uploader": username,
			"CRC32":    crc32Checksum,
			"CRC64":    crc64Checksum,
			"SHA1":     sha1Checksum,
			"SHA256":   sha256Checksum,
			"BLAKE2sp": blake2spChecksum,
		}

		// Save metadata
		err = h.fileService.AddMetadata(dstPath, metadata)
		if err != nil {
			http.Error(w, "Error saving metadata", http.StatusInternalServerError)
			return
		}

		// Check if the file is an HTML file and extract metadata
		if strings.HasSuffix(fileHeader.Filename, ".html") {
			err = h.fileService.ExtractMetadataFromHTML(dstPath)
			if err != nil {
				logger.Warningf("Error extracting metadata from HTML file: %v", err)
			}
		}

		logger.Infof("User %s uploaded file: %s", username, fileHeader.Filename)
	}

	http.Redirect(w, r, reqPath, http.StatusSeeOther)
}

// DeleteHandler обрабатывает запросы на удаление файлов и папок.
func (h *FileHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username, err := h.authService.GetSessionUsername(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
		logger.Infof("User %s deleted item: %s", username, item)
	}

	// Redirect to the current directory after deletion
	currentPath := r.FormValue("currentPath")
	http.Redirect(w, r, currentPath, http.StatusSeeOther)
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

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username, err := h.authService.GetSessionUsername(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	err = h.fileService.RenamePath(fullOldPath, fullNewPath)
	if err != nil {
		http.Error(w, "Error renaming item", http.StatusInternalServerError)
		return
	}

	logger.Infof("User %s renamed item from %s to %s", username, oldPath, newName)
	http.Redirect(w, r, filepath.Dir(oldPath), http.StatusSeeOther)
}

// MoveHandler обрабатывает запросы на перемещение файлов и папок.
func (h *FileHandler) MoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username, err := h.authService.GetSessionUsername(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	itemPathsJSON := r.FormValue("itemPaths")
	destinationPath := r.FormValue("destinationPath")

	var itemPaths []string
	err = json.Unmarshal([]byte(itemPathsJSON), &itemPaths)
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

		logger.Infof("User %s moved item from %s to %s", username, itemPath, destinationPath)
	}

	http.Redirect(w, r, destinationPath, http.StatusSeeOther)
}

func (h *FileHandler) FileMetadataHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	fullPath := h.fileService.GetFullPath(filePath)
	metaFilePath := filepath.Join(filepath.Dir(fullPath), "."+filepath.Base(fullPath)+".meta")
	metadataFile, err := os.Open(metaFilePath)
	if os.IsNotExist(err) {
		// Если файл метаданных отсутствует, возвращаем пустой объект
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{})
		return
	} else if err != nil {
		http.Error(w, "Error reading metadata", http.StatusInternalServerError)
		return
	}
	defer metadataFile.Close()

	var metadata map[string]string
	decoder := json.NewDecoder(metadataFile)
	if err := decoder.Decode(&metadata); err != nil {
		http.Error(w, "Error decoding metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

// RecalculateHashesHandler обрабатывает запросы на пересчет хеш-сумм.
func (h *FileHandler) RecalculateHashesHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	fullPath := h.fileService.GetFullPath(filePath)
	hashes, err := h.fileService.RecalculateHashes(fullPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error recalculating hashes: %v", err), http.StatusInternalServerError)
		return
	}

	// Update metadata with new hashes
	err = h.fileService.AddMetadata(fullPath, hashes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error updating metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hashes)
}

// SaveMetadataHandler обрабатывает запросы на сохранение метаданных.
func (h *FileHandler) SaveMetadataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username, err := h.authService.GetSessionUsername(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var metadata map[string]string
	err = json.NewDecoder(r.Body).Decode(&metadata)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	filePath, ok := metadata["FilePath"]
	if !ok {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	fullPath := h.fileService.GetFullPath(filePath)
	err = h.fileService.AddMetadata(fullPath, metadata)
	if err != nil {
		http.Error(w, "Error saving metadata", http.StatusInternalServerError)
		return
	}

	logger.Infof("User %s updated metadata for file: %s", username, filePath)
	w.WriteHeader(http.StatusOK)
}
