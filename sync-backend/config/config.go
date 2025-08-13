package config

import (
    "encoding/json"
    "os"
)

type Config struct {
    Port string `json:"port"`
}

var Cfg *Config

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    Cfg = &cfg
    return Cfg, nil
}