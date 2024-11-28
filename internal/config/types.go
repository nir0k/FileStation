package config

// Config - структура для конфигурации приложения
type Config struct {
	WebServer WebServer `yaml:"web-server"`
	Logging   Logging   `yaml:"logging"`
}

// WebServer - конфигурация веб-сервера
type WebServer struct {
	Port     		string `yaml:"port"`
	Protocol 		string `yaml:"protocol"`
	BaseDir  		string `yaml:"base_dir"`
	SSLCert  		string `yaml:"ssl_cert_file,omitempty"`
	SSLKey   		string `yaml:"ssl_key_file,omitempty"`
	Version         string `yaml:"version"`
}

// Logging - конфигурация логгирования
type Logging struct {
	LogFile     string `yaml:"log_file"`
	LogSeverity string `yaml:"log_severity"`
	LogMaxSize  int    `yaml:"log_max_size"`
	LogMaxFiles int    `yaml:"log_max_files"`
	LogMaxAge   int    `yaml:"log_max_age"`
}