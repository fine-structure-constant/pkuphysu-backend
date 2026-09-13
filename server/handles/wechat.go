package handles

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/utils"
	wechatclient "pkuphysu-backend/internal/wechat"

	"github.com/gin-gonic/gin"
)

const wechatMPName = "物院学生会"

func WechatPosts(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		utils.RespondError(c, http.StatusBadRequest, "InvalidLimit", err)
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		utils.RespondError(c, http.StatusBadRequest, "InvalidPage", err)
		return
	}
	articles, count, err := db.ListWechatArticles((page-1)*limit, limit, wechatMPName)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatArticleQueryFailed", err)
		return
	}
	data := make([]gin.H, 0, len(articles))
	for _, article := range articles {
		title := article.Title
		tag := "其它"
		if strings.HasPrefix(title, "【") {
			if end := strings.Index(title, "】"); end > 1 {
				tag = title[len("【"):end]
				title = strings.TrimSpace(title[end+len("】"):])
			}
		}
		description := strings.Split(article.Description, "\n")[0]
		data = append(data, gin.H{"id": article.ID, "title": title, "description": description, "author": article.Author, "cover_url": article.CoverURL, "mp_name": article.MpName, "url": article.URL, "publish_time": time.Unix(article.PublishTime, 0).In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05"), "tag": tag})
	}
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": data, "count": count})
}

func WechatStatus(c *gin.Context) {
	client, err := wechatclient.Default()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatInitFailed", err)
		return
	}
	if _, err := client.Token(); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "TokenExpired", err)
		return
	}
	utils.RespondSuccess(c, gin.H{"success": true})
}

func WechatQRCode(c *gin.Context) {
	client, err := wechatclient.Default()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatInitFailed", err)
		return
	}
	fingerprint := c.Query("fingerprint")
	switch c.Query("action") {
	case "getqrcode":
		data, contentType, err := client.GetQRCode(fingerprint)
		if err != nil {
			utils.RespondError(c, http.StatusBadGateway, "WechatQRCodeFailed", err)
			return
		}
		if contentType == "" {
			contentType = "image/png"
		}
		c.Data(http.StatusOK, contentType, data)
	case "ask":
		result, err := client.AskQRCode(fingerprint)
		if err != nil {
			utils.RespondError(c, http.StatusBadGateway, "WechatQRCodePollFailed", err)
			return
		}
		c.JSON(http.StatusOK, result)
	default:
		utils.RespondError(c, http.StatusBadRequest, "InvalidAction", nil)
	}
}

func WechatLogin(c *gin.Context) {
	client, err := wechatclient.Default()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatInitFailed", err)
		return
	}
	result, err := client.Login(c.Query("fingerprint"))
	if err != nil {
		utils.RespondError(c, http.StatusBadGateway, "WechatLoginFailed", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func WechatUpdatePosts(c *gin.Context) {
	begin, count, ok := wechatPageParams(c)
	if !ok {
		return
	}
	client, err := wechatclient.Default()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatInitFailed", err)
		return
	}
	articles, err := client.UpdateArticles(begin, count)
	if err != nil {
		utils.RespondError(c, http.StatusBadGateway, "WechatUpdateFailed", err)
		return
	}
	utils.RespondSuccess(c, articles)
}

// WechatAppmsgPublish keeps the legacy Flask proxy endpoint available for
// callers that need the parsed article list without the success envelope.
func WechatAppmsgPublish(c *gin.Context) {
	begin, count, ok := wechatPageParams(c)
	if !ok {
		return
	}
	client, err := wechatclient.Default()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatInitFailed", err)
		return
	}
	articles, err := client.UpdateArticles(begin, count)
	if err != nil {
		utils.RespondError(c, http.StatusBadGateway, "WechatUpdateFailed", err)
		return
	}
	c.JSON(http.StatusOK, articles)
}

func wechatPageParams(c *gin.Context) (int, int, bool) {
	begin, err := strconv.Atoi(c.DefaultQuery("begin", "0"))
	if err != nil || begin < 0 {
		utils.RespondError(c, http.StatusBadRequest, "InvalidBegin", err)
		return 0, 0, false
	}
	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil || count < 1 || count > 50 {
		utils.RespondError(c, http.StatusBadRequest, "InvalidCount", err)
		return 0, 0, false
	}
	return begin, count, true
}

func WechatCookieHealth(c *gin.Context) {
	client, err := wechatclient.Default()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "WechatInitFailed", err)
		return
	}
	expiry := client.CookieExpiry()
	if expiry < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"expire": -1})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expire": expiry})
}
