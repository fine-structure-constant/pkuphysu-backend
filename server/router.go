package server

import (
	"net/http"
	"path/filepath"
	"pkuphysu-backend/internal/config"
	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/logger"
	"pkuphysu-backend/server/handles"
	"pkuphysu-backend/server/middlewares"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

func Init(e *gin.Engine) {
	config.InitConfig()
	logger.Init()
	e.Use(gin.LoggerWithWriter(log.StandardLogger().Out))
	e.Use(gin.RecoveryWithWriter(log.StandardLogger().Out))
	db.InitDB()

	e.Use(middlewares.RateLimit())
	Cors(e)

	e.POST("/auth/login", handles.Login)
	e.POST("/auth/register", handles.Register)
	e.POST("/iaaa/login", handles.IaaaLogin)
	e.POST("/email/send", handles.SendVerificationEmail)
	e.POST("/email/verify", handles.VerifyEmail)

	e.GET("/static/*file", handles.StaticFile)
	e.POST("/files/upload", middlewares.Auth(), handles.UploadFile)
	e.GET("/files/*filename", handles.StaticFile)

	dba := e.Group("/dba", middlewares.Auth(), middlewares.AuthAdmin())
	dba.POST("/db-tables/create-all", handles.CreateAll)
	dba.GET("/db-tables", handles.ListTables)
	dba.GET("/db-tables/:table", handles.GetTableData)
	dba.DELETE("/db-tables/:table", handles.DeleteTableRecords)
	dba.PUT("/db-tables/:table", handles.UpsertTableRecords)
	dba.PATCH("/db-tables/:table", handles.UpsertTableRecords)
	dba.GET("/db-tables/migrate", handles.CheckMigration)
	dba.POST("/db-tables/migrate", handles.ExecuteMigration)

	e.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	e.GET("/auth/member-authorized", middlewares.Auth(), middlewares.AuthMember(), handles.MemberAuthorized)

	g := e.Group("", middlewares.Auth())

	g.GET("/user/me", handles.CurrentUser)
	g.DELETE("/user/me", handles.DeleteUser)
	g.PUT("/user/me", handles.UpdateUserInfo)
	g.POST("/user/avatar", handles.UploadAvatar)
	g.GET("/user/posts", handles.MyPosts)
	g.GET("/user/comments", handles.MyComments)
	e.GET("/user/avatar/:id", handles.GetAvatar)
	g.POST("/auth/change-password", handles.ChangePassword)
	g.GET("/users", handles.ListUsers)
	g.GET("/admins", handles.ListAdmins)

	e.GET("/forum/posts", middlewares.OptionalAuth(), handles.GetPosts)
	e.GET("/forum/posts/:id", middlewares.OptionalAuth(), handles.GetPost)
	e.GET("/forum/comments/:id", middlewares.OptionalAuth(), handles.GetComments)
	g.POST("/forum/comments", handles.SubmitComment)
	g.POST("/forum/posts", handles.SubmitPost)
	g.PUT("/forum/posts/:id", handles.UpdatePost)
	g.DELETE("/forum/posts/:id", handles.DeleteOwnPost)
	g.PUT("/forum/comments/:id", handles.UpdateComment)
	g.DELETE("/forum/comments/:id", handles.DeleteOwnComment)
	g.POST("/forum/reports/:id", handles.ReportPost)
	g.GET("/forum/follow", handles.GetFollowedPosts)
	g.POST("/forum/follow/:id", handles.FollowPost)
	g.POST("/forum/like/:id", handles.LikePost)
	g.POST("/forum/comment/like/:id", handles.LikeComment)
	e.GET("/forum/tags", handles.GetTags)
	e.GET("/wechat/posts", handles.WechatPosts)
	e.POST("/markdown/preview", handles.Markdown)
	g.GET("/forum/posts/raw/:id", handles.GetRawPost)
	g.GET("/forum/comments/raw/:id", handles.GetRawComment)

	admin := e.Group("/admin", middlewares.Auth(), middlewares.AuthAdmin())
	admin.DELETE("/forum/posts/:id", handles.DeletePostByID)
	admin.DELETE("/forum/comments/:id", handles.DeleteCommentByID)
	admin.POST("/user/create", handles.CreateUser)
	admin.PATCH("/users/:id", handles.UpdateUserAccess)
	admin.DELETE("/users/:id", handles.DeleteUserByID)
	admin.GET("/forum/reports", handles.ListReports)
	admin.POST("/forum/reports/:id/resolve", handles.ResolveReport)

	wechatAdmin := e.Group("/wechat", middlewares.Auth(), middlewares.AuthAdmin())
	wechatAdmin.GET("/", handles.WechatStatus)
	wechatAdmin.GET("/scanloginqrcode", handles.WechatQRCode)
	wechatAdmin.GET("/login", handles.WechatLogin)
	wechatAdmin.GET("/update-posts", handles.WechatUpdatePosts)
	wechatAdmin.POST("/update-posts", handles.WechatUpdatePosts)
	wechatAdmin.GET("/cgi-bin/appmsgpublish", handles.WechatAppmsgPublish)
	wechatAdmin.GET("/check-health", handles.WechatCookieHealth)

	// Production SPA assets are emitted here by the frontend Vite build.
	e.Static("/assets", "./data/static/app/assets")
	e.StaticFile("/hero.jpg", "./data/static/app/hero.jpg")
	serveFrontendPage := func(page string) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.File(filepath.Join("./data/static/app", page, "index.html"))
		}
	}
	for route, page := range map[string]string{
		"/forum":            "forum",
		"/official_account": "official_account",
		"/login":            "login",
		"/usr":              "usr",
		"/archive":          "archive",
		"/about":            "about",
		"/articles":         "articles",
		"/console":          "console",
	} {
		e.GET(route, serveFrontendPage(page))
		e.GET(route+"/", serveFrontendPage(page))
	}
	e.GET("/", func(c *gin.Context) { c.File("./data/static/app/index.html") })
	e.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet && strings.Contains(c.GetHeader("Accept"), "text/html") {
			c.File("./data/static/app/index.html")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "errid": "NotFound", "message": "route not found"})
	})

}

func Cors(e *gin.Engine) {
	conf := cors.DefaultConfig()
	conf.AllowOrigins = config.Conf.Cors.AllowOrigins
	conf.AllowHeaders = config.Conf.Cors.AllowHeaders
	conf.AllowMethods = config.Conf.Cors.AllowMethods
	e.Use(cors.New(conf))
}
