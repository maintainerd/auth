package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var RedisClient *redis.Client

const smsTemplateCacheTTL = 15 * time.Minute

type smsTemplateRow struct {
	Message string `gorm:"column:message"`
}

type cachedSMSTemplate struct {
	Message string `json:"message"`
}

func smsTemplateCacheKey(tenantID int64, name string) string {
	return fmt.Sprintf("sms_tpl:%d:%s", tenantID, name)
}

func RenderTemplate(db *gorm.DB, templateName string, tenantID int64, data any) (string, error) {
	if db == nil {
		return "", fmt.Errorf("db is nil")
	}

	row, err := fetchSMSTemplate(db, templateName, tenantID)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("sms").Parse(row.Message)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fetchSMSTemplate(db *gorm.DB, name string, tenantID int64) (*smsTemplateRow, error) {
	ctx := context.Background()
	cacheKey := smsTemplateCacheKey(tenantID, name)

	if RedisClient != nil {
		cached, err := RedisClient.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var ct cachedSMSTemplate
			if json.Unmarshal(cached, &ct) == nil {
				return &smsTemplateRow{Message: ct.Message}, nil
			}
		}
	}

	var row smsTemplateRow
	err := db.Table("sms_templates").
		Select("message").
		Where("name = ? AND tenant_id = ? AND status = ? AND deleted_at IS NULL", name, tenantID, "active").
		First(&row).Error
	if err != nil {
		return nil, err
	}

	if RedisClient != nil {
		ct := cachedSMSTemplate{Message: row.Message}
		if b, err := json.Marshal(ct); err == nil {
			_ = RedisClient.Set(ctx, cacheKey, b, smsTemplateCacheTTL).Err()
		}
	}

	return &row, nil
}

func InvalidateTemplateCache(tenantID int64, name string) {
	if RedisClient == nil {
		return
	}
	_ = RedisClient.Del(context.Background(), smsTemplateCacheKey(tenantID, name)).Err()
}
