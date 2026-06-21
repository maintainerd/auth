package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var RedisClient *redis.Client

const emailTemplateCacheTTL = 15 * time.Minute

type emailTemplateRow struct {
	Subject   string  `gorm:"column:subject"`
	BodyHTML  string  `gorm:"column:body_html"`
	BodyPlain *string `gorm:"column:body_plain"`
}

type RenderedEmail struct {
	Subject   string
	BodyHTML  string
	BodyPlain string
}

type cachedTemplate struct {
	Subject   string  `json:"subject"`
	BodyHTML  string  `json:"body_html"`
	BodyPlain *string `json:"body_plain,omitempty"`
}

func emailTemplateCacheKey(tenantID int64, name string) string {
	return fmt.Sprintf("email_tpl:%d:%s", tenantID, name)
}

func RenderTemplate(db *gorm.DB, templateName string, tenantID int64, data any) (*RenderedEmail, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	row, err := fetchTemplate(db, templateName, tenantID)
	if err != nil {
		return nil, err
	}

	htmlTmpl, err := template.New("html").Parse(row.BodyHTML)
	if err != nil {
		return nil, err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return nil, err
	}

	var plainStr string
	if row.BodyPlain != nil && *row.BodyPlain != "" {
		plainTmpl, err := template.New("plain").Parse(*row.BodyPlain)
		if err != nil {
			return nil, err
		}
		var plainBuf bytes.Buffer
		if err := plainTmpl.Execute(&plainBuf, data); err != nil {
			return nil, err
		}
		plainStr = plainBuf.String()
	}

	return &RenderedEmail{
		Subject:   row.Subject,
		BodyHTML:  htmlBuf.String(),
		BodyPlain: plainStr,
	}, nil
}

func fetchTemplate(db *gorm.DB, name string, tenantID int64) (*emailTemplateRow, error) {
	ctx := context.Background()
	cacheKey := emailTemplateCacheKey(tenantID, name)

	if RedisClient != nil {
		cached, err := RedisClient.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var ct cachedTemplate
			if json.Unmarshal(cached, &ct) == nil {
				return &emailTemplateRow{
					Subject:   ct.Subject,
					BodyHTML:  ct.BodyHTML,
					BodyPlain: ct.BodyPlain,
				}, nil
			}
		}
	}

	var row emailTemplateRow
	err := db.Table("email_templates").
		Select("subject, body_html, body_plain").
		Where("name = ? AND tenant_id = ? AND status = ? AND deleted_at IS NULL", name, tenantID, "active").
		First(&row).Error
	if err != nil {
		return nil, err
	}

	if RedisClient != nil {
		ct := cachedTemplate{Subject: row.Subject, BodyHTML: row.BodyHTML, BodyPlain: row.BodyPlain}
		if b, err := json.Marshal(ct); err == nil {
			_ = RedisClient.Set(ctx, cacheKey, b, emailTemplateCacheTTL).Err()
		}
	}

	return &row, nil
}

func InvalidateTemplateCache(tenantID int64, name string) {
	if RedisClient == nil {
		return
	}
	_ = RedisClient.Del(context.Background(), emailTemplateCacheKey(tenantID, name)).Err()
}
