package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type LocalDirConfig struct {
	GroupID  string `yaml:"group_id"`
	LocalDir string `yaml:"local_dir"`
}

func SaveLocalDir(lc LocalDirConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(lc.LocalDir, 0700); err != nil {
		return err
	}
	configDir := filepath.Join(home, ".ferry", "config", "group")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(lc)
	if err != nil {
		return err
	}
	path := filepath.Join(configDir, lc.GroupID+".yml")
	return os.WriteFile(path, data, 0600)
}

func GetLocalDirs() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	// map of group id to local_dir
	localDirMap := make(map[string]string)
	configDir := filepath.Join(home, ".ferry", "config", "group")
	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if !file.IsDir() {
			data, err := os.ReadFile(filepath.Join(configDir, file.Name()))
			if err != nil {
				continue
			}
			var cfg LocalDirConfig
			err = yaml.Unmarshal(data, &cfg)
			if err != nil {
				continue
			}
			localDirMap[cfg.GroupID] = cfg.LocalDir
		}
	}
	return localDirMap, nil
}
