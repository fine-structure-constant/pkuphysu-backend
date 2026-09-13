package db

import (
	"pkuphysu-backend/internal/model"
	"pkuphysu-backend/internal/utils"
	"time"
)

func UpdateForumPost(post *model.ForumPost) error {
	if post.Type == model.ForumPostCanvasV1 {
		canvas, err := utils.DecodeCanvas(post.Content)
		if err != nil {
			return err
		}
		post.ContentHTML = utils.MarkdownToHtml(canvas.DescriptionMarkdown)
		post.ContentText = utils.MarkdownToText(canvas.DescriptionMarkdown)
	} else {
		post.Type = model.ForumPostMarkdown
		post.ContentHTML = utils.MarkdownToHtml(post.Content)
		post.ContentText = utils.MarkdownToText(post.Content)
	}
	return db.Model(post).Select("content", "content_html", "content_text", "type").Updates(post).Error
}

func UpdateForumComment(comment *model.ForumComment) error {
	comment.ContentHTML = utils.MarkdownToHtml(comment.Content)
	comment.ContentText = utils.MarkdownToText(comment.Content)
	return db.Model(comment).Select("content", "content_html", "content_text").Updates(comment).Error
}

func CreateForumReport(report *model.ForumReport) error { return db.Create(report).Error }

func GetOpenForumReports() ([]model.ForumReport, error) {
	var reports []model.ForumReport
	err := db.Where("resolved_at IS NULL").Order("id DESC").Find(&reports).Error
	return reports, err
}

func ResolveForumReport(id uint) error {
	now := time.Now()
	return db.Model(&model.ForumReport{}).Where("id = ?", id).Update("resolved_at", &now).Error
}

func UpdateUserAccess(id uint, role int, disabled bool) error {
	return db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{"role": role, "disabled": disabled}).Error
}

func GetPostsByUser(userID uint) ([]model.ForumPost, error) {
	var posts []model.ForumPost
	err := db.Preload("User").Preload("Tags").Where("user_id = ?", userID).Order("id DESC").Find(&posts).Error
	return posts, err
}

func GetCommentsByUser(userID uint) ([]model.ForumComment, error) {
	var comments []model.ForumComment
	err := db.Preload("User").Where("user_id = ?", userID).Order("id DESC").Find(&comments).Error
	return comments, err
}
