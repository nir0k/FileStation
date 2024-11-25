package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileService отвечает за операции с файлами и директориями.
type FileService struct {
	baseDir string
	authService *AuthService
}

// NewFileService создает новый экземпляр FileService.
func NewFileService(baseDir string, authService *AuthService) *FileService {
	return &FileService{
		baseDir: baseDir,
		authService: authService,
	}
}

// Добавьте метод для получения `AuthService`, если требуется
func (fs *FileService) GetAuthService() *AuthService {
	return fs.authService
}

// GetFullPath возвращает полный путь к файлу или директории.
func (fs *FileService) GetFullPath(relativePath string) string {
	return filepath.Join(fs.baseDir, filepath.Clean(relativePath))
}

// SaveFile сохраняет файл из потока ввода.
func (fs *FileService) SaveFile(dstPath string, src io.Reader) error {
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, src)
	return err
}

// Delete удаляет файл или директорию (рекурсивно).
func (fs *FileService) Delete(path string) error {
	return fs.DeletePath(path)
}

// CreateFolder создает директорию по указанному пути.
func (fs *FileService) CreateFolder(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// Rename переименовывает файл или директорию.
func (fs *FileService) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// Move перемещает файл или директорию.
func (fs *FileService) Move(src, dest string) error {
	return fs.MovePath(src, dest)
}

// IsDir проверяет, является ли указанный путь директорией.
func (fs *FileService) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// ListDirectory возвращает список содержимого директории.
func (fs *FileService) ListDirectory(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

// GetFileInfo возвращает информацию о файле.
func (fs *FileService) GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// DeletePath удаляет файл или директорию (рекурсивно).
func (fs *FileService) DeletePath(path string) error {
	return os.RemoveAll(path)
}

// RenamePath переименовывает файл или директорию.
func (fs *FileService) RenamePath(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// MovePath перемещает файл или директорию в новое место.
func (fs *FileService) MovePath(src, dest string) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating destination directory: %w", err)
	}
	return os.Rename(src, dest)
}

// AddFileToZip добавляет файл в ZIP-архив.
func (fs *FileService) AddFileToZip(zipWriter *zip.Writer, fullPath, relPath string) error {
	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil // Пропускаем директории
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = relPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// CreateZipArchive создает ZIP-архив из списка файлов.
func (fs *FileService) CreateZipArchive(w io.Writer, files []string) error {
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, file := range files {
		fullPath := fs.GetFullPath(file)
		relPath := strings.TrimPrefix(file, "/")
		if err := fs.AddFileToZip(zipWriter, fullPath, relPath); err != nil {
			return fmt.Errorf("error adding file to ZIP: %w", err)
		}
	}

	return nil
}

// FormatReadableSize возвращает читаемый размер файла.
func (fs *FileService) FormatReadableSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// GetModificationTimes возвращает карту дат изменения для файлов в директории.
func (fs *FileService) GetModificationTimes(path string) (map[string]time.Time, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	modTimes := make(map[string]time.Time)
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		modTimes[file.Name()] = info.ModTime()
	}

	return modTimes, nil
}

func (fs *FileService) AddMetadata(filePath string, metadata map[string]string) error {
    metaFilePath := filePath + ".meta" // Сохраняем метаданные в отдельном файле
    file, err := os.Create(metaFilePath)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    return encoder.Encode(metadata)
}