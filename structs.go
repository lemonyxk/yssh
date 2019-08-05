package main

type ServerConfigList []ServerConfig

type ServerConfig struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Host     string `json:"host"`
	Port     int `json:"port"`
	Password string `json:"password"`
}