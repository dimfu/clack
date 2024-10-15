package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Key     string `json:"key"`
	Tempo   int64  `json:"tempo"`
	Timesig string `json:"timesig"`
}

type ConfigManager struct {
	Config     []Config
	ConfigPath string
	File       *os.File
	FileInfo   os.FileInfo
}

func NewConfigManager() (*ConfigManager, error) {
	filePath := UserHomeDir() + ".clack.json"
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &ConfigManager{
		File:       f,
		ConfigPath: filePath,
		FileInfo:   fileInfo,
		Config:     []Config{},
	}, nil
}

func (cm *ConfigManager) IsFileNotEmpty() bool {
	return cm.FileInfo.Size() > 0
}

func (cm *ConfigManager) LoadConfig() error {
	if cm.IsFileNotEmpty() {
		if err := json.NewDecoder(cm.File).Decode(&cm.Config); err != nil {
			return err
		}
	}
	return nil
}

func (cm *ConfigManager) WriteConfig(cfg Config) error {
	cm.Config = append(cm.Config, cfg)
	newConf, err := json.Marshal(cm.Config)
	if err != nil {
		return err
	}

	err = os.WriteFile(cm.ConfigPath, newConf, 0644)
	if err != nil {
		return err
	}

	return nil
}
