package database

import (
	"github.com/Behyna/sms-services/smsgateway/internal/config"
	"github.com/Behyna/sms-services/smsgateway/pkg/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewConnection(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	return mysql.NewConnection(&cfg.Database, logger)
}
