package config

import (
	"12-08-2025/cmd/model"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// GetConfig reads port, temp-folder/archive folder(will be created if don't exist) and valid extensions from .env file. If any is not specified or env-file not found, it sets default value.
func GetConfig() model.Config {
	config := model.Config{ValidExt: []string{"pdf", "jpg"}, AppPort: 8080, TmpDir: "tmp", ArchDir: "archive"}
	if err := godotenv.Load(); err != nil {
		log.Printf("Failed to read .env-file: %v", err)
		log.Printf("Continue with default values: \n-port: '%d' \n-tmp-directory: '%s' \n-archive-directory: %s \n-valid exts: '%v'", config.AppPort, config.TmpDir, config.ArchDir, config.ValidExt)
		return config
	}

	//читаем порт для запуска
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		log.Printf("APP_PORT is not set in env. Continue with default port: '%d'", config.AppPort)
	} else {
		if port, err := strconv.Atoi(appPort); err != nil {
			log.Printf("Failed to parse APP_PORT value from env.: %s \nContinue with default port '%d'", err, config.AppPort)
		} else {
			config.AppPort = port
		}
	}

	//читаем название временной папки
	tmpDir := os.Getenv("TMP_DIRECTORY")
	if tmpDir == "" {
		log.Printf("TMP_DIRECTORY is not set in env. Continue with default value: '%s'", config.TmpDir)
	} else {
		config.TmpDir = tmpDir
	}

	//читаем название папки для складывания архивов
	archDir := os.Getenv("ARCH_DIRECTORY")
	if archDir == "" {
		log.Printf("ARCH_DIRECTORY is not set in env. Continue with default value: '%s'", config.ArchDir)
	} else {
		config.ArchDir = archDir
	}

	//создание папки для складывания АРХИВОВ, если её нет
	if err := os.MkdirAll(config.ArchDir, 0755); err != nil {
		log.Fatalf("Failed to create ARCHIVE directory: %v", err)
	} else {
		log.Println("Directory for storing archives created.")
	}
	//создание ВРЕМЕННОЙ папки для скачиваний, если её нет
	if err := os.MkdirAll(config.TmpDir, 0755); err != nil {
		log.Fatalf("Failed to create TEMP directory: %v", err)
	} else {
		log.Println("Temp-directory for storing downloads created.")
	}

	//читаем валидные расширения
	extsStr := os.Getenv("VALID_EXTENTIONS")
	if extsStr == "" {
		log.Printf("VALID_EXTENTIONS is not set in env. Continue with default exts: '%v'", config.ValidExt)
	} else {
		var arrExt []string
		err := json.Unmarshal([]byte(extsStr), &arrExt)
		if err != nil {
			log.Printf("Failed to parse VALID_EXTENTIONS value from env.: %s \nContinue with default exts '%v'", err, config.ValidExt)
			return config
		}
		config.ValidExt = arrExt
	}
	return config
}
