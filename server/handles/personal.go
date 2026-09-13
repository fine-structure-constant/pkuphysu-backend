package handles

import (
	"github.com/gin-gonic/gin"
	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/model"
	"pkuphysu-backend/internal/utils"
)

func MyPosts(c *gin.Context) {
	user := c.MustGet("CurrentUser").(*model.User)
	posts, err := db.GetPostsByUser(user.ID)
	if err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	result := make([]gin.H, 0, len(posts))
	for _, post := range posts {
		tags := make([]string, len(post.Tags))
		for i, tag := range post.Tags {
			tags[i] = tag.Name
		}
		result = append(result, gin.H{"id": post.ID, "text": post.ContentHTML, "type": post.Type, "timestamp": post.CreatedAt.Unix(), "follownum": post.Follownum, "likenum": post.Likenum, "reply": post.Reply, "tags": tags, "userid": user.ID, "username": user.Username})
	}
	utils.RespondSuccess(c, result)
}

func MyComments(c *gin.Context) {
	user := c.MustGet("CurrentUser").(*model.User)
	comments, err := db.GetCommentsByUser(user.ID)
	if err != nil {
		utils.RespondError(c, 500, "ServerError", err)
		return
	}
	result := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		result = append(result, gin.H{"cid": comment.ID, "pid": comment.PostID, "text": comment.ContentHTML, "timestamp": comment.CreatedAt.Unix(), "userid": user.ID, "username": user.Username, "likenum": comment.Likenum})
	}
	utils.RespondSuccess(c, result)
}
