package db

import (
	"pkuphysu-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ListWechatArticles(offset, limit int, mpName string) ([]model.WechatArticle, int64, error) {
	var articles []model.WechatArticle
	var count int64
	query := db.Model(&model.WechatArticle{}).Where("mp_name = ?", mpName)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("publish_time DESC, id ASC").Offset(offset).Limit(limit).Find(&articles).Error; err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

func UpsertWechatArticles(articles []model.WechatArticle) error {
	if len(articles) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "description", "author", "cover_url", "mp_name", "publish_time",
		}),
	}).Create(&articles).Error
}

func GetWechatCookies() ([]model.WechatCookie, error) {
	var cookies []model.WechatCookie
	return cookies, db.Find(&cookies).Error
}

func ReplaceWechatCookies(cookies []model.WechatCookie) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.WechatCookie{}).Error; err != nil {
			return err
		}
		if len(cookies) == 0 {
			return nil
		}
		return tx.Create(&cookies).Error
	})
}
