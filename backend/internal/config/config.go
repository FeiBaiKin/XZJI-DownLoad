package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config 全局配置，对应 config.yaml 结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port           int      `mapstructure:"port"`
	AllowedOrigins []string `mapstructure:"allowed_origins"` // CORS 允许的来源，空则允许 *
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

// ConfigPath 配置文件路径（backend/config.yaml）
var ConfigPath string

func init() {
	ConfigPath = filepath.Join(baseDir(), "config.yaml")
}

// baseDir 返回 backend 根目录，优先工作目录（go run 兼容）
func baseDir() string {
	wd, err := os.Getwd()
	if err == nil {
		dir := wd
		for i := 0; i < 3; i++ {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

// Load 加载配置文件。文件不存在时返回默认空配置（安装前阶段）。
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(ConfigPath)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("XZJI")
	v.AutomaticEnv()

	cfg := &Config{}
	if _, err := os.Stat(ConfigPath); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		if err := v.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 4000
	}
	return cfg, nil
}

// SaveConfig 将安装向导提交的配置写入 config.yaml
func SaveConfig(dbHost string, dbPort int, dbUser, dbPassword, dbName string, allowedOrigins []string, jwtSecret string) error {
	v := viper.New()
	v.SetConfigFile(ConfigPath)
	v.SetConfigType("yaml")

	v.Set("server.port", 4000)
	if len(allowedOrigins) > 0 {
		v.Set("server.allowed_origins", allowedOrigins)
	}
	v.Set("database.host", dbHost)
	v.Set("database.port", dbPort)
	v.Set("database.user", dbUser)
	v.Set("database.password", dbPassword)
	v.Set("database.name", dbName)
	if jwtSecret == "" {
		jwtSecret = GenerateJWTSecret()
	}
	v.Set("jwt.secret", jwtSecret)

	if dir := filepath.Dir(ConfigPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}
	return v.WriteConfig()
}

// GenerateJWTSecret 生成 32 字节随机密钥（hex 编码）
func GenerateJWTSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(fmt.Sprintf("xzji-%d", os.Getpid())))
	}
	return hex.EncodeToString(b)
}

// DSN 拼接 GORM/MySQL 连接串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

// DSNWithoutDB 拼接不含库名的连接串（用于安装时建库）
func (c *DatabaseConfig) DSNWithoutDB() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port)
}
