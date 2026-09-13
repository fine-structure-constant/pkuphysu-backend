package model

type WechatArticle struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description"`
	Author      string `json:"author"`
	CoverURL    string `json:"cover_url" gorm:"column:cover_url"`
	MpName      string `json:"mp_name" gorm:"column:mp_name;index;not null"`
	URL         string `json:"url" gorm:"column:url;uniqueIndex;not null"`
	PublishTime int64  `json:"publish_time" gorm:"column:publish_time;index;not null"`
}

type WechatCookie struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"not null;uniqueIndex:uq_wechat_cookie"`
	Value    string `gorm:"not null"`
	Domain   string `gorm:"not null;uniqueIndex:uq_wechat_cookie"`
	Path     string `gorm:"not null;uniqueIndex:uq_wechat_cookie"`
	Expires  int64
	Secure   bool
	HTTPOnly bool `gorm:"column:http_only"`
}
