package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

var session = struct {
	sync.RWMutex
	cookie string
}{}

type navData struct {
	IsLogin   bool   `json:"isLogin"`
	Mid       int64  `json:"mid"`
	Uname     string `json:"uname"`
	Face      string `json:"face"`
	VipStatus int    `json:"vipStatus"`
	VipType   int    `json:"vipType"`
}

func handleQRStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var out apiResponse[struct {
		URL       string `json:"url"`
		QRCodeKey string `json:"qrcode_key"`
	}]
	if err := getJSON(r.Context(), "https://passport.bilibili.com/x/passport-login/web/qrcode/generate", "", &out); err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	if out.Code != 0 || out.Data.QRCodeKey == "" {
		jsonError(w, 502, "B站无法生成登录二维码")
		return
	}
	png, err := qrcode.Encode(out.Data.URL, qrcode.Medium, 256)
	if err != nil {
		jsonError(w, 500, "二维码生成失败")
		return
	}
	writeJSON(w, map[string]string{"key": out.Data.QRCodeKey, "image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)})
}

func handleQRPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" || len(key) > 128 {
		jsonError(w, 400, "二维码标识无效")
		return
	}
	endpoint := "https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=" + url.QueryEscape(key)
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	setHeaders(req, currentCookie())
	resp, err := apiClient.Do(req)
	if err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	var out apiResponse[struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}]
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		jsonError(w, 502, "扫码状态解析失败")
		return
	}
	if out.Code != 0 {
		jsonError(w, 502, "B站返回："+out.Message)
		return
	}
	if out.Data.Code == 0 {
		cookie := mergeCookies(currentCookie(), resp.Cookies())
		if cookieValue(cookie, "SESSDATA") == "" {
			jsonError(w, 502, "登录成功但未收到会话凭据")
			return
		}
		session.Lock()
		session.cookie = cookie
		session.Unlock()
	}
	writeJSON(w, map[string]any{"code": out.Data.Code, "message": out.Data.Message})
}

func mergeCookies(existing string, incoming []*http.Cookie) string {
	values := map[string]string{}
	for _, part := range strings.Split(existing, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			values[kv[0]] = kv[1]
		}
	}
	for _, c := range incoming {
		if c.Value != "" {
			values[c.Name] = c.Value
		}
	}
	parts := make([]string, 0, len(values))
	for k, v := range values {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

var imageClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if !allowedImageURL(req.URL) {
			return fmt.Errorf("图片重定向到非 B 站域名")
		}
		if len(via) > 5 {
			return fmt.Errorf("图片重定向过多")
		}
		return nil
	},
}

func handleImage(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	} else if strings.HasPrefix(raw, "http://") {
		raw = "https://" + strings.TrimPrefix(raw, "http://")
	}
	u, err := url.Parse(raw)
	if err != nil || !allowedImageURL(u) {
		jsonError(w, 400, "只允许加载 B 站图片")
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 BiliGreen/"+version)
	resp, err := imageClient.Do(req)
	if err != nil {
		http.Error(w, "图片加载失败", 502)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
		http.Error(w, "图片服务器返回异常", 502)
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || len(data) == 8<<20 {
		http.Error(w, "图片过大或读取失败", 502)
		return
	}
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, bytes.NewReader(data))
}

func allowedImageURL(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return u.Scheme == "https" && (host == "hdslb.com" || strings.HasSuffix(host, ".hdslb.com"))
}

type videoCard struct {
	Aid      int64  `json:"aid"`
	BVID     string `json:"bvid"`
	Title    string `json:"title"`
	Pic      string `json:"pic"`
	Duration any    `json:"duration"`
	Author   string `json:"author,omitempty"`
	Owner    struct {
		Name string `json:"name"`
	} `json:"owner,omitempty"`
}

type favoriteFolder struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	MediaCount int    `json:"media_count"`
}

func currentCookie() string {
	session.RLock()
	defer session.RUnlock()
	return session.cookie
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cookie := currentCookie()
		if cookie == "" {
			writeJSON(w, navData{})
			return
		}
		var out apiResponse[navData]
		if err := getJSON(r.Context(), "https://api.bilibili.com/x/web-interface/nav", cookie, &out); err != nil {
			jsonError(w, 502, err.Error())
			return
		}
		writeJSON(w, out.Data)
	case http.MethodPost:
		var in struct{ Cookie string }
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			jsonError(w, 400, "请求格式错误")
			return
		}
		cookie := strings.TrimSpace(in.Cookie)
		if cookie == "" {
			jsonError(w, 400, "Cookie 不能为空")
			return
		}
		var out apiResponse[navData]
		if err := getJSON(r.Context(), "https://api.bilibili.com/x/web-interface/nav", cookie, &out); err != nil {
			jsonError(w, 502, err.Error())
			return
		}
		if out.Code != 0 || !out.Data.IsLogin {
			jsonError(w, 401, "Cookie 无效或已过期")
			return
		}
		session.Lock()
		session.cookie = cookie
		session.Unlock()
		writeJSON(w, out.Data)
	case http.MethodDelete:
		session.Lock()
		session.cookie = ""
		session.Unlock()
		writeJSON(w, map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleFeed(w http.ResponseWriter, r *http.Request) {
	fresh := positiveInt(r.URL.Query().Get("fresh"), 1)
	var out apiResponse[struct {
		Item []videoCard `json:"item"`
	}]
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/web-interface/index/top/feed/rcmd?ps=20&fresh_idx=%d&fresh_idx_1h=%d&fresh_type=3", fresh, fresh)
	if err := getJSON(r.Context(), endpoint, currentCookie(), &out); err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	if out.Code != 0 {
		jsonError(w, 502, "B站返回："+out.Message)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, out.Data.Item)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	keyword := strings.TrimSpace(r.URL.Query().Get("q"))
	if keyword == "" {
		jsonError(w, 400, "请输入关键词")
		return
	}
	page := positiveInt(r.URL.Query().Get("page"), 1)
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/web-interface/wbi/search/type?search_type=video&keyword=%s&page=%d", url.QueryEscape(keyword), page)
	var out apiResponse[struct {
		Result []videoCard `json:"result"`
	}]
	if err := getJSON(r.Context(), endpoint, currentCookie(), &out); err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	if out.Code != 0 {
		jsonError(w, 502, "B站返回："+out.Message)
		return
	}
	writeJSON(w, out.Data.Result)
}

func handleFavorites(w http.ResponseWriter, r *http.Request) {
	login, err := requireLogin(r.Context())
	if err != nil {
		jsonError(w, 401, err.Error())
		return
	}
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/v3/fav/folder/created/list-all?up_mid=%d", login.Mid)
	var out apiResponse[struct {
		List []favoriteFolder `json:"list"`
	}]
	if err := getJSON(r.Context(), endpoint, currentCookie(), &out); err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	if out.Code != 0 {
		jsonError(w, 502, "B站返回："+out.Message)
		return
	}
	writeJSON(w, out.Data.List)
}

func handleFavoriteItems(w http.ResponseWriter, r *http.Request) {
	if _, err := requireLogin(r.Context()); err != nil {
		jsonError(w, 401, err.Error())
		return
	}
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		jsonError(w, 400, "缺少收藏夹")
		return
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 0 {
		limit = 20
	}
	if limit == 0 {
		limit = 5000
	}
	items := make([]videoCard, 0, min(limit, 100))
	for page := 1; len(items) < limit; page++ {
		endpoint := fmt.Sprintf("https://api.bilibili.com/x/v3/fav/resource/list?media_id=%s&pn=%d&ps=20&order=mtime&type=0&tid=0&platform=web", url.QueryEscape(folder), page)
		var out apiResponse[struct {
			Medias  []videoCard `json:"medias"`
			HasMore bool        `json:"has_more"`
		}]
		if err := getJSON(r.Context(), endpoint, currentCookie(), &out); err != nil {
			jsonError(w, 502, err.Error())
			return
		}
		if out.Code != 0 {
			jsonError(w, 502, "B站返回："+out.Message)
			return
		}
		remaining := limit - len(items)
		if len(out.Data.Medias) > remaining {
			out.Data.Medias = out.Data.Medias[:remaining]
		}
		items = append(items, out.Data.Medias...)
		if !out.Data.HasMore || len(out.Data.Medias) == 0 {
			break
		}
	}
	writeJSON(w, items)
}

func requireLogin(ctx context.Context) (navData, error) {
	cookie := currentCookie()
	if cookie == "" {
		return navData{}, fmt.Errorf("请先登录")
	}
	var out apiResponse[navData]
	if err := getJSON(ctx, "https://api.bilibili.com/x/web-interface/nav", cookie, &out); err != nil {
		return navData{}, err
	}
	if out.Code != 0 || !out.Data.IsLogin {
		return navData{}, fmt.Errorf("登录已失效，请重新登录")
	}
	return out.Data, nil
}

func applyFavoriteAction(ctx context.Context, cookie, action string, sourceFolder, targetFolder, aid int64) error {
	csrf := cookieValue(cookie, "bili_jct")
	if csrf == "" {
		return fmt.Errorf("Cookie 中缺少 bili_jct，不能修改收藏夹")
	}
	form := url.Values{"resources": {fmt.Sprintf("%d:2", aid)}, "csrf": {csrf}}
	endpoint := "https://api.bilibili.com/x/v3/fav/resource/batch-del"
	form.Set("media_id", strconv.FormatInt(sourceFolder, 10))
	if action == "move" {
		if targetFolder == 0 || targetFolder == sourceFolder {
			return fmt.Errorf("请选择另一个目标收藏夹")
		}
		endpoint = "https://api.bilibili.com/x/v3/fav/resource/move"
		form.Del("media_id")
		form.Set("src_media_id", strconv.FormatInt(sourceFolder, 10))
		form.Set("tar_media_id", strconv.FormatInt(targetFolder, 10))
	} else if action != "remove" {
		return fmt.Errorf("未知的收藏夹操作")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	setHeaders(req, cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out apiResponse[json.RawMessage]
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return err
	}
	if out.Code != 0 {
		return fmt.Errorf("B站返回：%s", out.Message)
	}
	return nil
}

func cookieValue(cookie, name string) string {
	for _, part := range strings.Split(cookie, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 && kv[0] == name {
			return kv[1]
		}
	}
	return ""
}

func positiveInt(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
