package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	SSLMode  string `mapstructure:"sslmode"`
	TimeZone string `mapstructure:"timezone"`
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
func SaveConfig(dbHost string, dbPort int, dbUser, dbPassword, dbName, dbSSLMode, dbTimeZone string, allowedOrigins []string, jwtSecret string) error {
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
	v.Set("database.sslmode", dbSSLMode)
	v.Set("database.timezone", dbTimeZone)
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

// SetAllowedOrigins 定点更新 config.yaml 的 server.allowed_origins 字段，
// 保留其余配置不变。用于激活时自动回写探测到的站点地址（域名自动获取场景）。
func SetAllowedOrigins(origins []string) error {
	v := viper.New()
	v.SetConfigFile(ConfigPath)
	v.SetConfigType("yaml")
	// 文件不存在时 viper.WriteConfig 会报错，先确保存在
	if _, err := os.Stat(ConfigPath); err != nil {
		return fmt.Errorf("配置文件不存在: %w", err)
	}
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	v.Set("server.allowed_origins", origins)
	return v.WriteConfig()
}

// DSN 拼接 PostgreSQL 连接串。URL 形式会安全转义用户名、密码和数据库名。
func (c *DatabaseConfig) DSN() string {
	return postgresDSN(c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode, c.TimeZone)
}

// DSNWithoutDB 连接 PostgreSQL 的默认维护库，用于安装时创建目标库。
func (c *DatabaseConfig) DSNWithoutDB() string {
	return postgresDSN(c.Host, c.Port, c.User, c.Password, "postgres", c.SSLMode, c.TimeZone)
}

func postgresDSN(host string, port int, user, password, database, sslMode, timeZone string) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   database,
	}
	q := u.Query()
	if sslMode == "" {
		sslMode = "disable"
	}
	q.Set("sslmode", sslMode)
	if timeZone == "" {
		timeZone = "Asia/Shanghai"
	}
	q.Set("TimeZone", timeZone)
	u.RawQuery = strings.ReplaceAll(q.Encode(), "%2F", "/")
	return u.String()
}
