package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"pkuphysu-backend/internal/config"
	"pkuphysu-backend/internal/db"
	"pkuphysu-backend/internal/model"
)

const (
	defaultBaseURL = "https://mp.weixin.qq.com"
	defaultMPName  = "物院学生会"
)

var fingerprintPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type capturedCookie struct {
	cookie http.Cookie
}

type captureTransport struct {
	base    http.RoundTripper
	capture func(*http.Request, []*http.Cookie)
}

func (t captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err == nil {
		t.capture(req, resp.Cookies())
	}
	return resp, err
}

type Client struct {
	mu      sync.Mutex
	baseURL string
	http    *http.Client
	cookies map[string]capturedCookie
}

var (
	defaultClient *Client
	defaultOnce   sync.Once
	defaultErr    error
)

func Default() (*Client, error) {
	defaultOnce.Do(func() {
		defaultClient, defaultErr = NewClient(defaultBaseURL)
	})
	return defaultClient, defaultErr
}

func NewClient(baseURL string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &Client{baseURL: strings.TrimRight(baseURL, "/"), cookies: make(map[string]capturedCookie)}
	c.http = &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
		Transport: captureTransport{
			base:    http.DefaultTransport,
			capture: c.captureCookies,
		},
	}
	if err := c.loadCookies(); err != nil {
		return nil, fmt.Errorf("load wechat cookies: %w", err)
	}
	return c, nil
}

func cookieKey(domain, path, name string) string {
	return domain + "\x00" + path + "\x00" + name
}

func (c *Client) captureCookies(req *http.Request, cookies []*http.Cookie) {
	for _, item := range cookies {
		domain := item.Domain
		if domain == "" {
			domain = req.URL.Hostname()
		}
		path := item.Path
		if path == "" {
			path = "/"
		}
		key := cookieKey(domain, path, item.Name)
		if item.MaxAge < 0 {
			delete(c.cookies, key)
			continue
		}
		copy := *item
		copy.Domain = domain
		copy.Path = path
		c.cookies[key] = capturedCookie{cookie: copy}
	}
}

func encryptionKey() [32]byte {
	return sha256.Sum256([]byte(config.Conf.JwtSecret + ":wechat-cookie:v1"))
}

func encryptValue(value string) (string, error) {
	key := encryptionKey()
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	data := gcm.Seal(nonce, nonce, []byte(value), nil)
	return "enc:v1:" + base64.RawStdEncoding.EncodeToString(data), nil
}

func decryptValue(value string) (string, error) {
	if !strings.HasPrefix(value, "enc:v1:") {
		return value, nil
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, "enc:v1:"))
	if err != nil {
		return "", err
	}
	key := encryptionKey()
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(data) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted cookie")
	}
	returnBytes, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	return string(returnBytes), err
}

func (c *Client) loadCookies() error {
	stored, err := db.GetWechatCookies()
	if err != nil {
		return err
	}
	grouped := make(map[string][]*http.Cookie)
	for _, item := range stored {
		value, err := decryptValue(item.Value)
		if err != nil {
			// A JWT-secret rotation makes the old cookie ciphertext unreadable.
			// Discard the stale login state so an administrator can scan again.
			return db.ReplaceWechatCookies(nil)
		}
		cookie := &http.Cookie{Name: item.Name, Value: value, Domain: item.Domain, Path: item.Path, Secure: item.Secure, HttpOnly: item.HTTPOnly}
		if item.Expires > 0 {
			cookie.Expires = time.Unix(item.Expires, 0)
		}
		c.cookies[cookieKey(item.Domain, item.Path, item.Name)] = capturedCookie{cookie: *cookie}
		host := strings.TrimPrefix(item.Domain, ".")
		grouped[host] = append(grouped[host], cookie)
	}
	for host, cookies := range grouped {
		c.http.Jar.SetCookies(&url.URL{Scheme: "https", Host: host, Path: "/"}, cookies)
	}
	return nil
}

func (c *Client) persistCookies() error {
	items := make([]model.WechatCookie, 0, len(c.cookies))
	for _, captured := range c.cookies {
		cookie := captured.cookie
		value, err := encryptValue(cookie.Value)
		if err != nil {
			return err
		}
		expires := int64(0)
		if !cookie.Expires.IsZero() {
			expires = cookie.Expires.Unix()
		}
		items = append(items, model.WechatCookie{Name: cookie.Name, Value: value, Domain: cookie.Domain, Path: cookie.Path, Expires: expires, Secure: cookie.Secure, HTTPOnly: cookie.HttpOnly})
	}
	return db.ReplaceWechatCookies(items)
}

func (c *Client) request(method, path string, form url.Values) (*http.Response, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/143 Safari/537.36")
	req.Header.Set("Origin", c.baseURL)
	req.Header.Set("Referer", c.baseURL+"/")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return c.http.Do(req)
}

func validateFingerprint(fingerprint string) error {
	if !fingerprintPattern.MatchString(fingerprint) {
		return errors.New("invalid browser fingerprint")
	}
	return nil
}

func (c *Client) GetQRCode(fingerprint string) ([]byte, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := validateFingerprint(fingerprint); err != nil {
		return nil, "", err
	}
	if resp, err := c.request(http.MethodGet, "/", nil); err == nil {
		resp.Body.Close()
	}
	common := url.Values{"fingerprint": {fingerprint}, "token": {""}, "lang": {"zh_CN"}, "f": {"json"}, "ajax": {"1"}}
	pre := cloneValues(common)
	pre.Set("action", "prelogin")
	if resp, err := c.request(http.MethodPost, "/cgi-bin/bizlogin", pre); err != nil {
		return nil, "", err
	} else {
		if err := requireSuccess(resp); err != nil {
			resp.Body.Close()
			return nil, "", err
		}
		resp.Body.Close()
	}
	start := cloneValues(common)
	start.Set("userlang", "zh_CN")
	start.Set("login_type", "3")
	start.Set("sessionid", strconv.FormatInt(time.Now().UnixMilli(), 10)+strconv.Itoa(1000+int(time.Now().UnixNano()%9000)))
	if resp, err := c.request(http.MethodPost, "/cgi-bin/bizlogin?action=startlogin", start); err != nil {
		return nil, "", err
	} else {
		if err := requireSuccess(resp); err != nil {
			resp.Body.Close()
			return nil, "", err
		}
		resp.Body.Close()
	}
	resp, err := c.request(http.MethodGet, "/cgi-bin/scanloginqrcode?action=getqrcode&random="+strconv.FormatInt(time.Now().UnixMilli(), 10)+"&login_appid=", nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if err := requireSuccess(resp); err != nil {
		return nil, "", err
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, "", errors.New("wechat did not return a QR-code image")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return data, contentType, err
}

func requireSuccess(resp *http.Response) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("wechat returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func cloneValues(source url.Values) url.Values {
	result := make(url.Values, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}

func (c *Client) AskQRCode(fingerprint string) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := validateFingerprint(fingerprint); err != nil {
		return nil, err
	}
	path := "/cgi-bin/scanloginqrcode?action=ask&fingerprint=" + url.QueryEscape(fingerprint) + "&token=&lang=zh_CN&f=json&ajax=1"
	return c.requestJSON(http.MethodGet, path, nil)
}

func (c *Client) Login(fingerprint string) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := validateFingerprint(fingerprint); err != nil {
		return nil, err
	}
	form := url.Values{"userlang": {"zh_CN"}, "redirect_url": {""}, "cookie_forbidden": {"0"}, "cookie_cleaned": {"1"}, "plugin_used": {"0"}, "login_type": {"3"}, "fingerprint": {fingerprint}, "token": {""}, "lang": {"zh_CN"}, "f": {"json"}, "ajax": {"1"}}
	result, err := c.requestJSON(http.MethodPost, "/cgi-bin/bizlogin?action=login", form)
	if err == nil {
		err = c.persistCookies()
	}
	return result, err
}

func (c *Client) requestJSON(method, path string, form url.Values) (map[string]any, error) {
	resp, err := c.request(method, path, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := requireSuccess(resp); err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tokenLocked()
}

func (c *Client) tokenLocked() (string, error) {
	resp, err := c.request(http.MethodGet, "/", nil)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if err := c.persistCookies(); err != nil {
		return "", err
	}
	token := resp.Request.URL.Query().Get("token")
	if token == "" || !strings.Contains(resp.Request.URL.Path, "home") {
		return "", errors.New("wechat login has expired")
	}
	return token, nil
}

func (c *Client) CookieExpiry() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	var expiry int64
	for _, item := range c.cookies {
		value := item.cookie.Expires.Unix()
		if item.cookie.Expires.IsZero() || value <= 0 {
			continue
		}
		if expiry == 0 || value < expiry {
			expiry = value
		}
	}
	if expiry == 0 {
		return -1
	}
	return expiry
}

func (c *Client) UpdateArticles(begin, count int) ([]model.WechatArticle, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	token, err := c.tokenLocked()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/cgi-bin/appmsgpublish?sub=list&begin=%d&count=%d&token=%s&lang=zh_CN&f=json", begin, count, url.QueryEscape(token))
	result, err := c.requestJSON(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	articles, err := parsePublishPage(result)
	if err != nil {
		return nil, err
	}
	if err := db.UpsertWechatArticles(articles); err != nil {
		return nil, err
	}
	return articles, nil
}

func parsePublishPage(result map[string]any) ([]model.WechatArticle, error) {
	raw, ok := result["publish_page"].(string)
	if !ok || raw == "" {
		return nil, errors.New("wechat response does not contain publish_page")
	}
	var page struct {
		PublishList []struct {
			PublishInfo string `json:"publish_info"`
		} `json:"publish_list"`
	}
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		return nil, fmt.Errorf("decode publish_page: %w", err)
	}
	articles := make([]model.WechatArticle, 0)
	for _, published := range page.PublishList {
		var info struct {
			AppMsgInfo []struct {
				Title      string `json:"title"`
				Digest     string `json:"digest"`
				Author     string `json:"author"`
				CoverURL   string `json:"cover"`
				ContentURL string `json:"content_url"`
				LineInfo   struct {
					SendTime int64 `json:"send_time"`
				} `json:"line_info"`
			} `json:"appmsg_info"`
		}
		if err := json.Unmarshal([]byte(published.PublishInfo), &info); err != nil {
			continue
		}
		for _, item := range info.AppMsgInfo {
			if item.ContentURL == "" || item.Title == "" {
				continue
			}
			articles = append(articles, model.WechatArticle{Title: item.Title, Description: item.Digest, Author: item.Author, CoverURL: item.CoverURL, MpName: defaultMPName, URL: item.ContentURL, PublishTime: item.LineInfo.SendTime})
		}
	}
	return articles, nil
}
