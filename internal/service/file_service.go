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
	"hash/crc32"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
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
		// Add directory entry to zip
		_, err := zipWriter.Create(relPath + "/")
		if err != nil {
			return err
		}

		// Recursively add directory contents
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			entryFullPath := filepath.Join(fullPath, entry.Name())
			entryRelPath := filepath.Join(relPath, entry.Name())
			if err := fs.AddFileToZip(zipWriter, entryFullPath, entryRelPath); err != nil {
				return err
			}
		}
		return nil
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

func (fs *FileService) AddMetadata(filePath string, newMetadata map[string]string) error {
    metaFilePath := filepath.Join(filepath.Dir(filePath), "." + filepath.Base(filePath) + ".meta")

    // Чтение существующих метаданных, если файл существует
    existingMetadata := make(map[string]string)
    if _, err := os.Stat(metaFilePath); err == nil {
        file, err := os.Open(metaFilePath)
        if (err != nil) {
            return fmt.Errorf("error opening metadata file: %w", err)
        }
        defer file.Close()

        decoder := json.NewDecoder(file)
        if err := decoder.Decode(&existingMetadata); err != nil {
            return fmt.Errorf("error decoding metadata file: %w", err)
        }
    }

    // Обновление существующих метаданных новыми данными
    for key, value := range newMetadata {
        existingMetadata[key] = value
    }

    // Запись обновленных метаданных обратно в файл
    file, err := os.Create(metaFilePath)
    if err != nil {
        return fmt.Errorf("error creating metadata file: %w", err)
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", " ") // Для удобства чтения
    if err := encoder.Encode(existingMetadata); err != nil {
        return fmt.Errorf("error encoding metadata: %w", err)
    }

    return nil
}

// RecalculateHashes пересчитывает хеш-суммы для файла.
func (fs *FileService) RecalculateHashes(filePath string) (map[string]string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, fmt.Errorf("error opening file: %w", err)
    }
    defer file.Close()

    crc32Hash := crc32.NewIEEE()
    md5Hash := md5.New()
    sha1Hash := sha1.New()
    sha256Hash := sha256.New()

    // MultiWriter to compute checksums
    writer := io.MultiWriter(crc32Hash, md5Hash, sha1Hash, sha256Hash)

    // Copy data from file to writer and compute hashes
    if _, err := io.Copy(writer, file); err != nil {
        return nil, fmt.Errorf("error calculating hashes: %w", err)
    }

    hashes := map[string]string{
        "CRC32":  strings.ToUpper(fmt.Sprintf("%x", crc32Hash.Sum32())),
        "MD5":    fmt.Sprintf("%x", md5Hash.Sum(nil)),
        "SHA1":   fmt.Sprintf("%x", sha1Hash.Sum(nil)),
        "SHA256": fmt.Sprintf("%x", sha256Hash.Sum(nil)),
    }

    return hashes, nil
}

