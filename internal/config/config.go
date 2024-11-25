package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

// LoadConfig - функция для загрузки конфигурации из файла
func LoadConfig(path string) (Config, error) {
	var config Config
	file, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(file, &config)
	return config, err
}
