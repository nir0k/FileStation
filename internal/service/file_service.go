package service

import (
	"archive/zip"
	"bufio"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/net/html"
	"hash/crc32"
	"hash/crc64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileService отвечает за операции с файлами и директориями.
type FileService struct {
	baseDir     string
	authService *AuthService
}

// NewFileService создает новый экземпляр FileService.
func NewFileService(baseDir string, authService *AuthService) *FileService {
	return &FileService{
		baseDir:     baseDir,
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
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	metaFilePath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".meta")
	if _, err := os.Stat(metaFilePath); err == nil {
		return os.Remove(metaFilePath)
	}
	return nil
}

// RenamePath переименовывает файл или директорию.
func (fs *FileService) RenamePath(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return err
	}
	oldMetaFilePath := filepath.Join(filepath.Dir(oldPath), "."+filepath.Base(oldPath)+".meta")
	newMetaFilePath := filepath.Join(filepath.Dir(newPath), "."+filepath.Base(newPath)+".meta")
	if _, err := os.Stat(oldMetaFilePath); err == nil {
		err = os.Rename(oldMetaFilePath, newMetaFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// MovePath перемещает файл или директорию в новое место.
func (fs *FileService) MovePath(src, dest string) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating destination directory: %w", err)
	}
	err := os.Rename(src, dest)
	if err != nil {
		return err
	}
	srcMetaFilePath := filepath.Join(filepath.Dir(src), "."+filepath.Base(src)+".meta")
	destMetaFilePath := filepath.Join(filepath.Dir(dest), "."+filepath.Base(dest)+".meta")
	if _, err := os.Stat(srcMetaFilePath); err == nil {
		return os.Rename(srcMetaFilePath, destMetaFilePath)
	}
	return nil
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

func (fs *FileService) ExtractMetadataFromReadme(dirPath string) (map[string]string, error) {
	readmePath := filepath.Join(dirPath, "README.md")
	file, err := os.Open(readmePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // README.md не существует
		}
		return nil, fmt.Errorf("ошибка при открытии README.md: %w", err)
	}
	defer file.Close()

	metadata := make(map[string]string)
	scanner := bufio.NewScanner(file)
	var inRDSSection bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "## RDS" {
			inRDSSection = true
			continue
		}
		if inRDSSection {
			if strings.HasPrefix(line, "## ") && line != "## RDS" {
				// Начался новый раздел, выходим из RDS секции
				break
			}
			if strings.HasPrefix(line, "- **") && strings.Contains(line, "**: `") && strings.HasSuffix(line, "`") {
				lineContent := strings.TrimPrefix(line, "- **")
				parts := strings.SplitN(lineContent, "**: `", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSuffix(parts[1], "`")
						metadata["RDS "+key] = strings.TrimSpace(value)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при чтении README.md: %w", err)
	}
	return metadata, nil
}

func (fs *FileService) AddMetadata(filePath string, newMetadata map[string]string) error {
	metaFilePath := filepath.Join(filepath.Dir(filePath), "."+filepath.Base(filePath)+".meta")

	// Чтение существующих метаданных, если файл существует
	existingMetadata := make(map[string]string)
	if _, err := os.Stat(metaFilePath); err == nil {
		file, err := os.Open(metaFilePath)
		if err != nil {
			return fmt.Errorf("ошибка при открытии файла метаданных: %w", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&existingMetadata); err != nil {
			return fmt.Errorf("ошибка при декодировании файла метаданных: %w", err)
		}
	}

	// Извлечение метаданных из README.md
	readmeMetadata, err := fs.ExtractMetadataFromReadme(filepath.Dir(filePath))
	if err != nil {
		return fmt.Errorf("ошибка при извлечении метаданных из README.md: %w", err)
	}

	// Объединение метаданных
	for key, value := range newMetadata {
		existingMetadata[key] = value
	}
	for key, value := range readmeMetadata {
		existingMetadata[key] = value
	}

	// Удаление ключа "File name" из метаданных
	delete(existingMetadata, "RDS File name")

	// Запись обновленных метаданных обратно в файл
	file, err := os.Create(metaFilePath)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла метаданных: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ") // Для удобства чтения
	if err := encoder.Encode(existingMetadata); err != nil {
		return fmt.Errorf("ошибка при кодировании метаданных: %w", err)
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
	crc64Hash := crc64.New(crc64.MakeTable(crc64.ECMA))
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()
	blake2spHash, err := blake2s.New256(nil)
	if err != nil {
		return nil, fmt.Errorf("error creating BLAKE2sp hash: %w", err)
	}

	writer := io.MultiWriter(crc32Hash, crc64Hash, sha1Hash, sha256Hash, blake2spHash)

	if _, err := io.Copy(writer, file); err != nil {
		return nil, fmt.Errorf("error calculating hashes: %w", err)
	}

	hashes := map[string]string{
		"CRC32":    strings.ToUpper(fmt.Sprintf("%x", crc32Hash.Sum32())),
		"CRC64":    strings.ToUpper(fmt.Sprintf("%x", crc64Hash.Sum(nil))),
		"SHA1":     fmt.Sprintf("%x", sha1Hash.Sum(nil)),
		"SHA256":   fmt.Sprintf("%x", sha256Hash.Sum(nil)),
		"BLAKE2sp": fmt.Sprintf("%x", blake2spHash.Sum(nil)),
	}

	return hashes, nil
}

func (fs *FileService) ExtractMetadataFromHTML(htmlFilePath string) error {
	fmt.Printf("Opening HTML file: %s\n", htmlFilePath)
	file, err := os.Open(htmlFilePath)
	if err != nil {
		return fmt.Errorf("error opening HTML file: %w", err)
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		return fmt.Errorf("error parsing HTML file: %w", err)
	}

	metadata := make(map[string]string)
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					text := strings.TrimSpace(c.Data)
					if strings.HasPrefix(text, "Дата проверки:") {
						metadata["Дата проверки"] = strings.TrimSpace(strings.TrimPrefix(text, "Дата проверки:"))
					} else if strings.HasPrefix(text, "Основание:") {
						metadata["RDS"] = strings.TrimSpace(strings.TrimPrefix(text, "Основание:"))
					} else if strings.HasPrefix(text, "Ссылка на RDS:") {
						metadata["Ссылка на RDS"] = strings.TrimSpace(strings.TrimPrefix(text, "Ссылка на RDS:"))
					} else if strings.HasPrefix(text, "CRC32:") {
						metadata["CRC32"] = strings.TrimSpace(strings.TrimPrefix(text, "CRC32:"))
					} else if strings.HasPrefix(text, "CRC64:") {
						metadata["CRC64"] = strings.TrimSpace(strings.TrimPrefix(text, "CRC64:"))
					} else if strings.HasPrefix(text, "SHA256:") {
						metadata["SHA256"] = strings.TrimSpace(strings.TrimPrefix(text, "SHA256:"))
					} else if strings.HasPrefix(text, "SHA1:") {
						metadata["SHA1"] = strings.TrimSpace(strings.TrimPrefix(text, "SHA1:"))
					} else if strings.HasPrefix(text, "BLAKE2sp:") {
						metadata["BLAKE2sp"] = strings.TrimSpace(strings.TrimPrefix(text, "BLAKE2sp:"))
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	readmePath := filepath.Join(filepath.Dir(htmlFilePath), "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		fmt.Printf("README.md does not exist, creating new one at %s\n", readmePath)
		return createReadme(readmePath, metadata)
	} else {
		fmt.Printf("README.md exists, updating at %s\n", readmePath)
		return updateReadme(readmePath, metadata)
	}
}

func createReadme(readmePath string, metadata map[string]string) error {
	fmt.Printf("Creating README.md at %s\n", readmePath)
	file, err := os.Create(readmePath)
	if (err != nil) {
		return fmt.Errorf("error creating README.md: %w", err)
	}
	defer file.Close()

	// Удаляем ключ "File name" перед записью в README.md
	delete(metadata, "File name")

	_, err = file.WriteString("## RDS\n")
	if (err != nil) {
		return fmt.Errorf("error writing to README.md: %w", err)
	}

	for key, value := range metadata {
		_, err := file.WriteString(fmt.Sprintf("- **%s**: `%s`\n", key, strings.TrimSpace(value)))
		if (err != nil) {
			return fmt.Errorf("error writing to README.md: %w", err)
		}
	}
	fmt.Printf("README.md created successfully at %s\n", readmePath)
	return nil
}

func updateReadme(readmePath string, metadata map[string]string) error {
	fmt.Printf("Updating README.md at %s\n", readmePath)
	file, err := os.OpenFile(readmePath, os.O_RDWR, 0644)
	if (err != nil) {
		return fmt.Errorf("error opening README.md: %w", err)
	}
	defer file.Close()

	existingContent, err := io.ReadAll(file)
	if (err != nil) {
		return fmt.Errorf("error reading README.md: %w", err)
	}

	existingLines := strings.Split(string(existingContent), "\n")
	updatedLines := make([]string, 0, len(existingLines))

	// Remove existing metadata lines and "## RDS" header
	for _, line := range existingLines {
		updated := false
		for key := range metadata {
			if strings.HasPrefix(line, "- **"+key+"**:") {
				updated = true
				break
			}
		}
		if line == "## RDS" {
			updated = true
		}
		if !updated {
			updatedLines = append(updatedLines, line)
		}
	}

	// Удаляем ключ "File name" перед обновлением README.md
	delete(metadata, "File name")

	// Add a section header for RDS
	updatedLines = append(updatedLines, "## RDS")

	// Append new metadata lines
	for key, value := range metadata {
		updatedLines = append(updatedLines, fmt.Sprintf("- **%s**: `%s`", key, strings.TrimSpace(value)))
	}

	file.Truncate(0)
	file.Seek(0, 0)
	for _, line := range updatedLines {
		_, err := file.WriteString(line + "\n")
		if (err != nil) {
			return fmt.Errorf("error writing to README.md: %w", err)
		}
	}
	fmt.Printf("README.md updated successfully at %s\n", readmePath)
	return nil
}