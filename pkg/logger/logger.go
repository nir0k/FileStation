// Пакет logger предоставляет настраиваемую утилиту для логирования в Go с поддержкой различных уровней логов,
// форматов, вывода в консоль и ротации логов. Пакет включает функции для логирования сообщений на уровнях TRACE, DEBUG, INFO, WARNING, ERROR и FATAL.
// Поддерживаются как текстовый, так и JSON формат вывода, а также настройка ротации файлов логов с заданием максимального размера, количества резервных копий и периода хранения.
// Логгер может выводить сообщения как в файл, так и в консоль с возможностью задания минимального уровня логирования для каждого типа вывода.
// В случае ошибки инициализации логгера, сообщение об ошибке будет выведено в консоль.
//
// Основные возможности:
//   - Логирование на различных уровнях (TRACE, DEBUG, INFO, WARNING, ERROR, FATAL).
//   - Поддержка форматов вывода: стандартный текстовый и JSON.
//   - Настройка уровней логирования для файла и консоли (можно задать числом или строкой).
//   - Опциональная ротация логов с возможностью сжатия старых файлов.
//   - Поддержка цветного вывода в консоль для удобства визуального восприятия.
//
// Пример использования:
//    config := logger.LogConfig{
//        FilePath: "./logs/app.log",
//        Format: "standard",
//        FileLevel: "debug",
//        ConsoleLevel: "info",
//        ConsoleOutput: true,
//        EnableRotation: true,
//        RotationConfig: logger.RotationConfig{
//            MaxSize: 10,
//            MaxBackups: 5,
//            MaxAge: 30,
//            Compress: true,
//        },
//    }
//
//    err := logger.InitLogger(config)
//    if err != nil {
//        fmt.Println("Ошибка инициализации логгера:", err)
//        return
//    }
//    logger.Info("Пример информационного сообщения")
package logger

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "time"

    "github.com/fatih/color"
    "github.com/natefinch/lumberjack"
)

// Глобальная переменная для экземпляра логгера
var logInstance *Logger

// InitLogger инициализирует логгер и сохраняет экземпляр в глобальной переменной logInstance.
//
// Аргументы:
//   - config (LogConfig): Конфигурация логгера с настройками уровня логирования, формата, вывода в файл и ротации.
//
// Возвращает:
//   - error: Ошибка, если инициализация не удалась, иначе nil.
func InitLogger(config LogConfig) error {
    var err error
    logInstance, err = NewLogger(config)
    if err != nil {
        // Вывод сообщения об ошибке в консоль
        fmt.Println("Logger initialization failed:", err)
    }
    return err
}

// LogConfig представляет настройки конфигурации для логгера.
type LogConfig struct {
    FilePath       string         // Полный путь к файлу логов.
    Format         string         // Формат логов: "standard" или "json".
    FileLevel      interface{}    // Уровень логов для вывода в файл: может быть строкой или числом.
    ConsoleLevel   interface{}    // Уровень логов для вывода в консоль: может быть строкой или числом.
    ConsoleOutput  bool           // Выводить ли логи в консоль.
    EnableRotation bool           // Включить ли ротацию логов.
    RotationConfig RotationConfig // Настройки для ротации логов.
}

// RotationConfig содержит настройки для ротации логов.
type RotationConfig struct {
    MaxSize    int  // Максимальный размер в мегабайтах перед ротацией логов.
    MaxBackups int  // Максимальное количество старых файлов логов для хранения.
    MaxAge     int  // Максимальное количество дней для хранения старых файлов логов.
    Compress   bool // Сжимать ли старые файлы логов.
}

// Logger представляет настраиваемый логгер с различными опциями конфигурации.
type Logger struct {
    FileLogger      *log.Logger
    ConsoleLogger   *log.Logger
    Config          LogConfig
    FileLogLevel    int
    ConsoleLogLevel int
    LogLevelMap     map[string]int
}

// setDefaults устанавливает значения по умолчанию для конфигурации логгера.
func setDefaults(config *LogConfig) {
    if config.Format == "" {
        config.Format = "standard"
    }
    if config.FileLevel == nil {
        config.FileLevel = "warning"
    }
    if config.ConsoleLevel == nil {
        config.ConsoleLevel = "warning"
    }
    if config.RotationConfig.MaxSize == 0 {
        config.RotationConfig.MaxSize = 10 // 10 МБ
    }
    if config.RotationConfig.MaxBackups == 0 {
        config.RotationConfig.MaxBackups = 7 // 7 резервных копий
    }
    if config.RotationConfig.MaxAge == 0 {
        config.RotationConfig.MaxAge = 30 // 30 дней
    }
}

// NewLogger создает и возвращает новый экземпляр Logger с указанной конфигурацией.
//
// Аргументы:
//   - config (LogConfig): Конфигурация логгера, включающая параметры уровня логирования, вывода и ротации.
//
// Возвращает:
//   - (*Logger): Указатель на новый экземпляр Logger.
//   - error: Ошибка, если конфигурация недействительна или файл логов недоступен.
func NewLogger(config LogConfig) (*Logger, error) {
    // Устанавливаем значения по умолчанию
    setDefaults(&config)

    l := &Logger{
        Config: config,
        LogLevelMap: map[string]int{
            "trace":   5,
            "debug":   4,
            "info":    3,
            "warning": 2,
            "error":   1,
            "fatal":   0,
        },
    }

    // Функция для получения числового значения уровня логирования
    getLogLevel := func(level interface{}) (int, error) {
        switch v := level.(type) {
        case string:
            logLevel, ok := l.LogLevelMap[strings.ToLower(v)]
            if !ok {
                return 0, fmt.Errorf("invalid log level: %s", v)
            }
            return logLevel, nil
        case int:
            if v < 0 {
                return 0, nil // уровень "fatal" для значений меньше 0
            } else if v > 5 {
                return 5, nil // уровень "trace" для значений больше 5
            }
            return v, nil
        default:
            return 0, fmt.Errorf("invalid type for log level: %T", v)
        }
    }

    // Устанавливаем уровни логирования для файла и консоли
    fileLevel, err := getLogLevel(config.FileLevel)
    if err != nil {
        fmt.Println("Invalid file log level:", err)
        return nil, fmt.Errorf("invalid file log level: %v", err)
    }
    l.FileLogLevel = fileLevel

    consoleLevel, err := getLogLevel(config.ConsoleLevel)
    if err != nil {
        fmt.Println("Invalid console log level:", err)
        return nil, fmt.Errorf("invalid console log level: %v", err)
    }
    l.ConsoleLogLevel = consoleLevel

    // Настройка логирования в файл, если указан путь
    if config.FilePath != "" {
        dir := filepath.Dir(config.FilePath)
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            return nil, fmt.Errorf("log directory does not exist: %s", dir)
        }

        var fileWriter io.Writer
        if config.EnableRotation {
            fileWriter = &lumberjack.Logger{
                Filename:   config.FilePath,
                MaxSize:    config.RotationConfig.MaxSize,
                MaxBackups: config.RotationConfig.MaxBackups,
                MaxAge:     config.RotationConfig.MaxAge,
                Compress:   config.RotationConfig.Compress,
            }
        } else {
            file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
            if err != nil {
                return nil, fmt.Errorf("failed to open log file: %v", err)
            }
            fileWriter = file
        }

        l.FileLogger = log.New(fileWriter, "", 0)
    } else {
        l.FileLogger = nil // Нет логгера файла, если FilePath не установлен
    }

    // Настройка вывода на консоль
    if config.ConsoleOutput {
        l.ConsoleLogger = log.New(os.Stdout, "", 0)
    }

    return l, nil
}

// log - это внутренний метод, который записывает сообщения с указанным уровнем и аргументами.
func (l *Logger) log(level string, v ...interface{}) {
    msgLevel, ok := l.LogLevelMap[level]
    if !ok {
        return
    }

    // Теперь проверка идет на "более высокий или равный" для вывода
    if msgLevel > l.FileLogLevel && msgLevel > l.ConsoleLogLevel {
        return
    }

    timestamp := time.Now().Format(time.RFC3339)
    pid := os.Getpid()

    // Получение информации о вызывающем
    _, file, line, ok := runtime.Caller(3)
    if !ok {
        file = "unknown"
        line = 0
    } else {
        file = trimPathToProject(file)
    }

    prefix := fmt.Sprintf("[%s] [PID: %d] [%s:%d] [%s] ", timestamp, pid, file, line, strings.ToUpper(level))

    var logEntry string

    if strings.ToLower(l.Config.Format) == "json" {
        logData := map[string]interface{}{
            "timestamp": timestamp,
            "level":     level,
            "pid":       pid,
            "file":      file,
            "line":      line,
            "message":   fmt.Sprint(v...),
        }
        jsonBytes, _ := json.Marshal(logData)
        logEntry = string(jsonBytes)
    } else {
        logEntry = prefix + fmt.Sprint(v...)
    }

    // Проверка уровня логирования для файла и консоли
    if l.FileLogger != nil && msgLevel <= l.FileLogLevel {
        l.FileLogger.Println(logEntry)
    }

    if l.Config.ConsoleOutput && msgLevel <= l.ConsoleLogLevel {
        colorFunc := color.New(color.FgWhite).SprintFunc()
        switch level {
        case "trace":
            colorFunc = color.New(color.FgCyan).SprintFunc()
        case "debug":
            colorFunc = color.New(color.FgBlue).SprintFunc()
        case "info":
            colorFunc = color.New(color.FgGreen).SprintFunc()
        case "warning":
            colorFunc = color.New(color.FgYellow).SprintFunc()
        case "error":
            colorFunc = color.New(color.FgRed).SprintFunc()
        case "fatal":
            colorFunc = color.New(color.FgHiRed).SprintFunc()
        }
        l.ConsoleLogger.Println(colorFunc(logEntry))
    }
}

// trimPathToProject обрезает путь к файлу до уровня проекта.
func trimPathToProject(filePath string) string {
    // Предполагаем, что каталог проекта - это тот, который содержит файл "go.mod"
    projectDir := findProjectDir()
    if projectDir == "" {
        return filepath.Base(filePath)
    }
    relPath, err := filepath.Rel(projectDir, filePath)
    if err != nil {
        return filepath.Base(filePath)
    }
    return relPath
}

// findProjectDir находит каталог проекта, ища файл "go.mod".
func findProjectDir() string {
    dir, err := os.Getwd()
    if err != nil {
        return ""
    }
    for {
        if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
            return dir
        }
        parentDir := filepath.Dir(dir)
        if parentDir == dir {
            break
        }
        dir = parentDir
    }
    return ""
}

// GetLoggerConfig возвращает текущую конфигурацию логгера.
//
// Возвращает:
//   - (LogConfig): Конфигурация логгера, используемая в logInstance.
func GetLoggerConfig() LogConfig {
    if logInstance != nil {
        return logInstance.Config
    }
    return LogConfig{}
}

// Пакетные функции-обёртки для методов логгера

// Trace записывает сообщение на уровне TRACE, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Trace(v ...interface{}) {
    if logInstance != nil {
        logInstance.Trace(v...)
    }
}

// Debug записывает сообщение на уровне DEBUG, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Debug(v ...interface{}) {
    if logInstance != nil {
        logInstance.Debug(v...)
    }
}

// Info записывает сообщение на уровне INFO, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Info(v ...interface{}) {
    if logInstance != nil {
        logInstance.Info(v...)
    }
}

// Warning записывает сообщение на уровне WARNING, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Warning(v ...interface{}) {
    if logInstance != nil {
        logInstance.Warning(v...)
    }
}

// Error записывает сообщение на уровне ERROR, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Error(v ...interface{}) {
    if logInstance != nil {
        logInstance.Error(v...)
    }
}

// Fatal записывает сообщение на уровне FATAL и завершает выполнение приложения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Fatal(v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatal(v...)
    }
}

// Tracef записывает форматированное сообщение на уровне TRACE, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func Tracef(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Tracef(format, v...)
    }
}

// Debugf записывает форматированное сообщение на уровне DEBUG, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func Debugf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Debugf(format, v...)
    }
}

// Infof записывает форматированное сообщение на уровне INFO, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func Infof(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Infof(format, v...)
    }
}

// Warningf записывает форматированное сообщение на уровне WARNING, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func Warningf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Warningf(format, v...)
    }
}

// Errorf записывает форматированное сообщение на уровне ERROR, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func Errorf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Errorf(format, v...)
    }
}

// Fatalf записывает форматированное сообщение на уровне FATAL и завершает выполнение приложения.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func Fatalf(format string, v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatalf(format, v...)
        os.Exit(1)
    }
}

// Traceln записывает сообщение на уровне TRACE с новой строкой, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Traceln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Traceln(v...)
    }
}

// Debugln записывает сообщение на уровне DEBUG с новой строкой, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Debugln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Debugln(v...)
    }
}

// Infoln записывает сообщение на уровне INFO с новой строкой, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Infoln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Infoln(v...)
    }
}

// Warningln записывает сообщение на уровне WARNING с новой строкой, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Warningln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Warningln(v...)
    }
}

// Errorln записывает сообщение на уровне ERROR с новой строкой, если уровень логирования позволяет вывод этого сообщения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Errorln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Errorln(v...)
    }
}

// Fatalln записывает сообщение на уровне FATAL с новой строкой и завершает выполнение приложения.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func Fatalln(v ...interface{}) {
    if logInstance != nil {
        logInstance.Fatalln(v...)
        os.Exit(1)
    }
}

// Методы экземпляра логгера

// Trace записывает сообщение на уровне TRACE.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Trace(v ...interface{}) {
    l.log("trace", v...)
}

// Debug записывает сообщение на уровне DEBUG.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Debug(v ...interface{}) {
    l.log("debug", v...)
}

// Info записывает сообщение на уровне INFO.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Info(v ...interface{}) {
    l.log("info", v...)
}

// Warning записывает сообщение на уровне WARNING.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Warning(v ...interface{}) {
    l.log("warning", v...)
}

// Error записывает сообщение на уровне ERROR.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Error(v ...interface{}) {
    l.log("error", v...)
}

// Fatal записывает сообщение на уровне FATAL и завершает приложение.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Fatal(v ...interface{}) {
    l.log("fatal", v...)
    os.Exit(1)
}

// Tracef записывает форматированное сообщение на уровне TRACE.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func (l *Logger) Tracef(format string, v ...interface{}) {
    l.log("trace", fmt.Sprintf(format, v...))
}

// Debugf записывает форматированное сообщение на уровне DEBUG.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func (l *Logger) Debugf(format string, v ...interface{}) {
    l.log("debug", fmt.Sprintf(format, v...))
}

// Infof записывает форматированное сообщение на уровне INFO.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func (l *Logger) Infof(format string, v ...interface{}) {
    l.log("info", fmt.Sprintf(format, v...))
}

// Warningf записывает форматированное сообщение на уровне WARNING.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func (l *Logger) Warningf(format string, v ...interface{}) {
    l.log("warning", fmt.Sprintf(format, v...))
}

// Errorf записывает форматированное сообщение на уровне ERROR.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func (l *Logger) Errorf(format string, v ...interface{}) {
    l.log("error", fmt.Sprintf(format, v...))
}

// Fatalf записывает форматированное сообщение на уровне FATAL и завершает приложение.
//
// Аргументы:
//   - format (string): Формат строки.
//   - v (...interface{}): Значения для форматирования сообщения.
func (l *Logger) Fatalf(format string, v ...interface{}) {
    l.log("fatal", fmt.Sprintf(format, v...))
    os.Exit(1)
}

// Traceln записывает сообщение на уровне TRACE с новой строкой.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Traceln(v ...interface{}) {
    l.log("trace", fmt.Sprintln(v...))
}

// Debugln записывает сообщение на уровне DEBUG с новой строкой.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Debugln(v ...interface{}) {
    l.log("debug", fmt.Sprintln(v...))
}

// Infoln записывает сообщение на уровне INFO с новой строкой.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Infoln(v ...interface{}) {
    l.log("info", fmt.Sprintln(v...))
}

// Warningln записывает сообщение на уровне WARNING с новой строкой.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Warningln(v ...interface{}) {
    l.log("warning", fmt.Sprintln(v...))
}

// Errorln записывает сообщение на уровне ERROR с новой строкой.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Errorln(v ...interface{}) {
    l.log("error", fmt.Sprintln(v...))
}

// Fatalln записывает сообщение на уровне FATAL с новой строкой и завершает приложение.
//
// Аргументы:
//   - v (...interface{}): Сообщение для логирования.
func (l *Logger) Fatalln(v ...interface{}) {
    l.log("fatal", fmt.Sprintln(v...))
    os.Exit(1)
}