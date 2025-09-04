package mysql

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type Config struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

func NewConnection(ctx context.Context, cfg Config, logger *zap.Logger) (db *gorm.DB, err error) {
	dsn := buildDSN(cfg)

	// TODO: gorm logger to read from config
	gormLogger := gormLogger.New(&zapWriter{logger: logger},
		gormLogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormLogger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		})

	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		logger.Error("Failed to connect to database",
			zap.Error(err),
			zap.String("host", cfg.Host),
			zap.String("database", cfg.Name),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get underlying DB", zap.Error(err))
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		logger.Error("Database ping failed", zap.Error(err))
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("Successfully connected to MySQL database",
		zap.String("host", cfg.Host),
		zap.String("database", cfg.Name),
	)

	return db.WithContext(ctx), nil
}

func buildDSN(cfg Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
}

type zapWriter struct {
	logger *zap.Logger
}

func (z *zapWriter) Printf(format string, args ...interface{}) {
	z.logger.Info(fmt.Sprintf(format, args...))
}
