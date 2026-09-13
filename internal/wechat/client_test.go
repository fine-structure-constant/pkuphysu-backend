package wechat

import (
	"testing"

	"pkuphysu-backend/internal/config"
)

func TestCookieEncryptionRoundTrip(t *testing.T) {
	previous := config.Conf
	config.Conf = &config.Config{JwtSecret: "test-secret"}
	t.Cleanup(func() { config.Conf = previous })

	encrypted, err := encryptValue("sensitive-cookie")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "sensitive-cookie" {
		t.Fatal("cookie was stored as plaintext")
	}
	decrypted, err := decryptValue(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "sensitive-cookie" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}
}

func TestParsePublishPage(t *testing.T) {
	result := map[string]any{
		"publish_page": `{"publish_list":[{"publish_info":"{\"appmsg_info\":[{\"title\":\"【通知】测试\",\"digest\":\"摘要\",\"author\":\"物院学生会\",\"cover\":\"https://example.com/cover.jpg\",\"content_url\":\"https://mp.weixin.qq.com/s/example\",\"line_info\":{\"send_time\":123}}]}"}]}`,
	}
	articles, err := parsePublishPage(result)
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 1 {
		t.Fatalf("expected one article, got %d", len(articles))
	}
	article := articles[0]
	if article.Title != "【通知】测试" || article.Description != "摘要" || article.Author != "物院学生会" || article.CoverURL != "https://example.com/cover.jpg" || article.URL != "https://mp.weixin.qq.com/s/example" || article.PublishTime != 123 {
		t.Fatalf("unexpected article: %#v", article)
	}
}

func TestValidateFingerprint(t *testing.T) {
	if err := validateFingerprint("abcDEF_123-xyz"); err != nil {
		t.Fatal(err)
	}
	if err := validateFingerprint("bad&token=other"); err == nil {
		t.Fatal("expected invalid fingerprint to be rejected")
	}
}
