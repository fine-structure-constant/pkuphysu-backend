package handles

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/model"
	"pkuphysu-backend/internal/utils"
)

type LoginReq struct {
	Username string `json:"username" binding:"required,min=1,max=50"`
	Password string `json:"password"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type RegisterReq struct {
	Username string `json:"username" binding:"required,alphanumunicode,min=1,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
}

func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", err)
		return
	}

	var user *model.User
	var err error

	if strings.Contains(req.Username, "@") {
		if !isValidPkuStudentEmail(req.Username) {
			utils.RespondError(c, http.StatusBadRequest, "invalid_email_format",
				fmt.Errorf("邮箱必须是@stu.pku.edu.cn域名，且前缀必须是学号"))
			return
		}

		stuid := extractStuidFromEmail(req.Username)
		user, err = db.GetUserByStuid(stuid)
	} else {
		user, err = db.GetUserByName(req.Username)
	}

	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "invalid_credentials", err)
		return
	}

	if err := user.ValidatePassword(req.Password); err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "invalid_credentials", err)
	} else {
		token, err := utils.GenerateToken(user)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "internal_server_error", err)
		} else {
			utils.RespondSuccess(c, gin.H{"token": token, "username": user.Username, "userid": user.ID})
		}
	}
}

// Register creates a normal account after the email verification record has
// been checked. The existing /email/verify endpoint is kept for compatibility
// with older clients that use email verification as an implicit registration.
func Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", err)
		return
	}

	if !isValidPkuStudentEmail(req.Email) {
		utils.RespondError(c, http.StatusBadRequest, "invalid_email_domain", fmt.Errorf("email must be a @stu.pku.edu.cn address with a numeric student ID"))
		return
	}

	if _, err := db.GetUserByName(req.Username); err == nil {
		utils.RespondError(c, http.StatusConflict, "username_already_exists", errors.New("username already exists"))
		return
	}

	stuid := extractStuidFromEmail(req.Email)
	if _, err := db.GetUserByStuid(stuid); err == nil {
		utils.RespondError(c, http.StatusConflict, "student_already_registered", errors.New("this student email has already been registered"))
		return
	}

	verification, err := db.GetEmailVerificationByEmail(req.Email)
	if err != nil || verification.Code != req.Code {
		utils.RespondError(c, http.StatusBadRequest, "invalid_verification_code", errors.New("invalid or expired verification code"))
		return
	}

	if err := db.MarkEmailVerificationAsUsed(req.Email); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed_to_consume_verification", err)
		return
	}

	user := &model.User{
		Username: req.Username,
		Stuid:    stuid,
		Role:     model.GENERAL,
		Verified: true,
	}
	user.SetPassword(req.Password)
	if err := db.CreateUser(user); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "user_creation_failed", err)
		return
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "token_generation_failed", err)
		return
	}

	utils.RespondSuccess(c, gin.H{"token": token, "username": user.Username, "userid": user.ID})
}

// MemberAuthorized is a small contract probe for member-only frontend routes.
func MemberAuthorized(c *gin.Context) {
	user := c.MustGet("CurrentUser").(*model.User)
	utils.RespondSuccess(c, gin.H{
		"authorized": true,
		"message":    "member authorized",
		"role":       user.Role,
		"userid":     user.ID,
	})
}

func ChangePassword(c *gin.Context) {
	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid_request", err)
		return
	}

	user := c.MustGet("CurrentUser").(*model.User)
	if user.PwdHash == "" && user.Salt == "" {
	} else {
		if err := user.ValidatePassword(req.OldPassword); err != nil {
			utils.RespondError(c, http.StatusUnauthorized, "invalid_current_password", err)
			return
		}
	}

	user.SetPassword(req.NewPassword)

	if err := db.UpdateUser(user); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed_to_update_password", err)
		return
	}

	utils.RespondSuccess(c, gin.H{"message": "password_updated_successfully"})
}
