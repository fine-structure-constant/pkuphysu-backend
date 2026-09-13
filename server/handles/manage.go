package handles

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/model"
	"pkuphysu-backend/internal/utils"
)

func UpdatePost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	post, err := db.GetForumPostByID(id)
	if err != nil {
		utils.RespondError(c, 404, "NotFound", err)
		return
	}
	user := c.MustGet("CurrentUser").(*model.User)
	if post.UserID != user.ID && !user.IsAdmin() {
		utils.RespondError(c, 403, "PermissionDenied", errors.New("only the author or an administrator can edit this post"))
		return
	}
	var req struct {
		Text   string               `json:"text"`
		Type   int                  `json:"type"`
		Canvas *utils.CanvasPayload `json:"canvas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, 400, "InvalidParams", err)
		return
	}
	if req.Type == model.ForumPostCanvasV1 {
		if req.Canvas == nil {
			utils.RespondError(c, 400, "InvalidCanvas", errors.New("canvas payload is required"))
			return
		}
		encoded, err := utils.EncodeCanvas(*req.Canvas)
		if err != nil {
			utils.RespondError(c, 400, "InvalidCanvas", err)
			return
		}
		post.Content = encoded
	} else if req.Type == model.ForumPostMarkdown {
		if strings.TrimSpace(req.Text) == "" || len([]byte(req.Text)) > 100*1024 {
			utils.RespondError(c, 400, "InvalidParams", errors.New("post must be between 1 byte and 100 KiB"))
			return
		}
		post.Content = req.Text
	} else {
		utils.RespondError(c, 400, "InvalidType", errors.New("unsupported post type"))
		return
	}
	post.Type = req.Type
	if err := db.UpdateForumPost(post); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "帖子已更新"})
}

func UpdateComment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	comment, err := db.GetForumCommentByID(uint(id))
	if err != nil {
		utils.RespondError(c, 404, "NotFound", err)
		return
	}
	user := c.MustGet("CurrentUser").(*model.User)
	if comment.UserID != user.ID && !user.IsAdmin() {
		utils.RespondError(c, 403, "PermissionDenied", errors.New("only the author or an administrator can edit this comment"))
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Text) == "" || len([]byte(req.Text)) > 50*1024 {
		utils.RespondError(c, 400, "InvalidParams", errors.New("comment must be between 1 byte and 50 KiB"))
		return
	}
	comment.Content = req.Text
	if err := db.UpdateForumComment(comment); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "评论已更新"})
}

func DeleteOwnPost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	post, err := db.GetForumPostByID(int(id))
	if err != nil {
		utils.RespondError(c, 404, "NotFound", err)
		return
	}
	user := c.MustGet("CurrentUser").(*model.User)
	if post.UserID != user.ID && !user.IsAdmin() {
		utils.RespondError(c, 403, "PermissionDenied", errors.New("only the author or an administrator can delete this post"))
		return
	}
	if err := db.DeleteForumPostByID(uint(id)); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "帖子已删除"})
}
func DeleteOwnComment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	comment, err := db.GetForumCommentByID(uint(id))
	if err != nil {
		utils.RespondError(c, 404, "NotFound", err)
		return
	}
	user := c.MustGet("CurrentUser").(*model.User)
	if comment.UserID != user.ID && !user.IsAdmin() {
		utils.RespondError(c, 403, "PermissionDenied", errors.New("only the author or an administrator can delete this comment"))
		return
	}
	if err := db.DeleteForumCommentByID(uint(id)); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "评论已删除"})
}

func ReportPost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" || len([]rune(req.Reason)) > 500 {
		utils.RespondError(c, 400, "InvalidReason", errors.New("reason must be between 1 and 500 characters"))
		return
	}
	user := c.MustGet("CurrentUser").(*model.User)
	if err := db.CreateForumReport(&model.ForumReport{PostID: uint(id), ReporterID: user.ID, Reason: req.Reason}); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "举报已提交"})
}
func ListReports(c *gin.Context) {
	reports, err := db.GetOpenForumReports()
	if err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, reports)
}
func ResolveReport(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	if err := db.ResolveForumReport(uint(id)); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "举报已处理"})
}

func UpdateUserAccess(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, 400, "InvalidID", err)
		return
	}
	var req struct {
		Role     int  `json:"role"`
		Disabled bool `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Role < model.GENERAL || req.Role > model.ADMIN {
		utils.RespondError(c, http.StatusBadRequest, "InvalidUserAccess", errors.New("invalid role or disabled state"))
		return
	}
	current := c.MustGet("CurrentUser").(*model.User)
	if current.ID == uint(id) && (req.Disabled || req.Role != model.ADMIN) {
		utils.RespondError(c, 400, "SelfLockout", errors.New("administrators cannot remove their own access"))
		return
	}
	if err := db.UpdateUserAccess(uint(id), req.Role, req.Disabled); err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "用户权限已更新"})
}
