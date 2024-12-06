// TODO: Добавить modal для логина
// TODO: расширить таблицу файлов

package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fileStation/internal/config"
	"fileStation/internal/handler"
	"fileStation/internal/service"
	"fileStation/pkg/logger"
)

var appVersion = "2.4.0"

//go:embed templates/* static/*
var embeddedFS embed.FS

var (
	indexTemplate *template.Template
	loginTemplate *template.Template
)

func loadTemplates() error {
	funcMap := template.FuncMap{
        "splitPath": func(p string) []string {
            return strings.Split(strings.Trim(p, "/"), "/")
        },
        "joinPath": func(base, elem string) string {
            if base == "/" {
                return "/" + elem
            }
            return base + "/" + elem
        },
        "getFileIcon": func(filename string) string {
            ext := strings.ToLower(filepath.Ext(filename))
            switch ext {
            case ".txt":
                return "description"
            case ".pdf":
                return "picture_as_pdf"
            case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
                return "image"
            case ".zip", ".rar", ".7z", ".tar", ".gz":
                return "archive"
            case ".doc", ".docx":
                return "description"
            case ".xls", ".xlsx":
                return "grid_on"
            case ".ppt", ".pptx":
                return "slideshow"
            case ".mp3", ".wav", ".aac":
                return "audiotrack"
            case ".mp4", ".avi", ".mov", ".mkv":
                return "movie"
            default:
                return "insert_drive_file"
            }
        },
        "getFileInfo": func(fullPath, name string) os.FileInfo {
            info, err := os.Stat(filepath.Join(fullPath, name))
            if err != nil {
                logger.Trace("Error getting file info:", err)
                return nil
            }
            return info
        },
        "readableSize": func(info os.FileInfo) string {
            if info == nil {
                return ""
            }
            size := info.Size()
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
        },
        "hasSuffix": strings.HasSuffix,
        "lower": strings.ToLower,
    }
    var err error
	indexTemplate, err = template.New("base.html").Funcs(funcMap).ParseFS(embeddedFS,
        "templates/base.html",
        "templates/navbar.html",
        "templates/index.html",
        "templates/footer.html",
    )
    if err != nil {
        return err
    }

    loginTemplate, err = template.ParseFS(embeddedFS,
        "templates/base.html",
        "templates/navbar.html",
        "templates/login.html",
		"templates/footer.html",
    )
    if err != nil {
        return err
    }

    return nil
}


func staticFileServer() http.Handler {
	staticFS, err := fs.Sub(embeddedFS, "static")
	if err != nil {
		logger.Fatalf("Failed to set up static file server: %v", err)
	}
	return http.FileServer(http.FS(staticFS))
}

func setup( cfg *config.Config, mux *http.ServeMux) {
	// Настройка логгера
	logConfig := logger.LogConfig{
		FilePath:      cfg.Logging.LogFile,
		Format:        "standard",
		FileLevel:     cfg.Logging.LogSeverity,
		ConsoleLevel:  "info",
		ConsoleOutput: true,
		EnableRotation: true,
		RotationConfig: logger.RotationConfig{
			MaxSize:    cfg.Logging.LogMaxSize,
			MaxBackups: cfg.Logging.LogMaxFiles,
			MaxAge:     cfg.Logging.LogMaxAge,
			Compress:   true,
		},
	}
	if err := logger.InitLogger(logConfig); err != nil {
		logger.Fatalf("Logger initialization failed: %v", err)
	}

	// Загружаем шаблоны
	if err := loadTemplates(); err != nil {
		logger.Fatalf("Failed to load templates: %v", err)
	}

	// Сервисы
	authService := service.NewAuthService()
	fileService := service.NewFileService(cfg.WebServer.BaseDir, authService)

	// Хендлеры
	authHandler := handler.NewAuthHandler(authService, loginTemplate, appVersion)
	fileHandler := handler.NewFileHandler(fileService, indexTemplate, authService, appVersion)
	helperHandler := handler.NewHelperHandler(fileService)

	// Статические файлы
	mux.Handle("/static/", http.StripPrefix("/static/", staticFileServer()))

	// Публичные маршруты	
	mux.HandleFunc("/login", authHandler.LoginHandler)
	mux.HandleFunc("/logout", authHandler.LogoutHandler)
	mux.HandleFunc("/check-session", authHandler.CheckSessionHandler)
	mux.HandleFunc("/", fileHandler.ServeFiles)
	mux.HandleFunc("/download", fileHandler.DownloadHandler)
	mux.HandleFunc("/dir-tree", helperHandler.DirTreeHandler)
	mux.HandleFunc("/list-folders", helperHandler.ListFoldersHandler)
	mux.HandleFunc("/file-metadata", fileHandler.FileMetadataHandler)
	mux.HandleFunc("/recalculate-hashes", fileHandler.RecalculateHashesHandler)

	// Защищённые маршруты
	mux.Handle("/upload", authHandler.Middleware(http.HandlerFunc(fileHandler.UploadHandler)))
	mux.Handle("/delete", authHandler.Middleware(http.HandlerFunc(fileHandler.DeleteHandler)))
	mux.Handle("/create-folder", authHandler.Middleware(http.HandlerFunc(fileHandler.CreateFolderHandler)))
	mux.Handle("/rename", authHandler.Middleware(http.HandlerFunc(fileHandler.RenameHandler)))
	mux.Handle("/move", authHandler.Middleware(http.HandlerFunc(fileHandler.MoveHandler)))
	mux.Handle("/save-metadata", authHandler.Middleware(http.HandlerFunc(fileHandler.SaveMetadataHandler)))

	

}

func main() {
	versionFlag := flag.Bool("version", false, "Display application version")
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("fileStation - Version %s\n", appVersion)
		os.Exit(0)
	}

	// Загрузка конфигурации
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create a new ServeMux
    mux := http.NewServeMux()

	// Set up the application with the custom mux
    setup(&cfg, mux)

	// Запуск сервера
	addr := ":" + cfg.WebServer.Port
	logger.Infof("Starting server on %s...", addr)

	if cfg.WebServer.Protocol == "https" {
		logger.Fatal(http.ListenAndServeTLS(
			addr,
			cfg.WebServer.SSLCert,
			cfg.WebServer.SSLKey,
			mux,
		))
	} else {
		logger.Fatal(http.ListenAndServe(addr, mux))
	}
}
