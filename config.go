package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

func (cm *ConfigManager) GetConfigByKey(key string) *Config {
	for _, config := range cm.Config {
		if config.Key == key {
			return &config
		}
	}
	return nil
}

func (cm *ConfigManager) WriteConfig() error {
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

func initConfigManager() (*ConfigManager, error) {
	cm, err := NewConfigManager()
	if err != nil {
		return nil, err
	}
	defer cm.File.Close()

	err = cm.LoadConfig()
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func CreateConf(cfg Config) error {
	cm, err := initConfigManager()
	if err != nil {
		return err
	}

	c := cm.GetConfigByKey(cfg.Key)
	if c != nil {
		return errors.New("this config is already exist")
	}

	cm.Config = append(cm.Config, cfg)
	if err := cm.WriteConfig(); err != nil {
		return err
	}

	return nil
}

func DeleteConfig(key string) error {
	cm, err := initConfigManager()
	if err != nil {
		return err
	}

	c := cm.GetConfigByKey(key)
	if c == nil {
		return fmt.Errorf("`%v` config not found", key)
	}

	for i, config := range cm.Config {
		if config.Key == c.Key {
			cm.Config = append(cm.Config[:i], cm.Config[i+1:]...)
		}
	}

	if err := cm.WriteConfig(); err != nil {
		return err
	}

	return nil
}
