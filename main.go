package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "0.5.0"

var apiClient = &http.Client{Timeout: 30 * time.Second}
var downloadClient = &http.Client{}
var appBaseDir = "."

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type videoInfo struct {
	BVID     string `json:"bvid"`
	Aid      int64  `json:"aid"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Pages    []page `json:"pages"`
}

type page struct {
	CID      int64  `json:"cid"`
	Page     int    `json:"page"`
	Part     string `json:"part"`
	Duration int    `json:"duration"`
}

type playData struct {
	Quality           int      `json:"quality"`
	Format            string   `json:"format"`
	AcceptQuality     []int    `json:"accept_quality"`
	AcceptDescription []string `json:"accept_description"`
	DURL              []durl   `json:"durl"`
	SupportFormats    []struct {
		Quality        int    `json:"quality"`
		NewDescription string `json:"new_description"`
		DisplayDesc    string `json:"display_desc"`
	} `json:"support_formats"`
	Dash *struct {
		Video []dashStream `json:"video"`
		Audio []dashStream `json:"audio"`
	} `json:"dash"`
}

type dashStream struct {
	ID        int    `json:"id"`
	BaseURL   string `json:"baseUrl"`
	BaseURL2  string `json:"base_url"`
	Bandwidth int64  `json:"bandwidth"`
	Codecs    string `json:"codecs"`
	Codecid   int    `json:"codecid"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

func (s dashStream) mediaURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return s.BaseURL2
}

type durl struct {
	Size      int64    `json:"size"`
	URL       string   `json:"url"`
	BackupURL []string `json:"backup_url"`
}

type task struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
	Error      string `json:"error,omitempty"`
	PostAction string `json:"post_action,omitempty"`
	Playable   bool   `json:"playable"`
	Path       string `json:"-"`
	Created    int64  `json:"created"`
	Duration   int    `json:"duration,omitempty"`
	PartInfo   string `json:"part_info,omitempty"`
	MediaType  string `json:"media_type,omitempty"`
}

var tasks = struct {
	sync.RWMutex
	m map[string]*task
}{m: make(map[string]*task)}

var downloadSlots = make(chan struct{}, 2)

func main() {
	if executable, err := os.Executable(); err == nil {
		appBaseDir = filepath.Dir(executable)
	}
	listen := flag.String("listen", "127.0.0.1:17890", "本地界面监听地址")
	noOpen := flag.Bool("no-open", false, "不自动打开浏览器")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveUI)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/tasks", handleTasks)
	mux.HandleFunc("/api/session", handleSession)
	mux.HandleFunc("/api/feed", handleFeed)
	mux.HandleFunc("/api/search", handleSearch)
	mux.HandleFunc("/api/favorites", handleFavorites)
	mux.HandleFunc("/api/favorite/items", handleFavoriteItems)
	mux.HandleFunc("/api/qr/start", handleQRStart)
	mux.HandleFunc("/api/qr/poll", handleQRPoll)
	mux.HandleFunc("/api/image", handleImage)
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"name": "BiliGreen", "version": version})
	})
	mux.HandleFunc("/api/qualities", handleQualities)
	mux.HandleFunc("/api/audio-qualities", handleAudioQualities)
	mux.HandleFunc("/api/media", handleMedia)
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/open-folder", handleOpenFolder)

	server := &http.Server{Addr: *listen, Handler: localOnly(mux), ReadHeaderTimeout: 5 * time.Second}
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
		go func() {
			time.Sleep(150 * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()
	})
	pageURL := "http://" + *listen
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		if existingInstance(pageURL) {
			_ = openBrowser(pageURL)
			return
		}
		message := "BiliGreen 启动失败：" + err.Error()
		writeStartupError(message)
		fmt.Fprintln(os.Stderr, message)
		return
	}
	if !*noOpen {
		go func() { time.Sleep(350 * time.Millisecond); _ = openBrowser(pageURL) }()
	}
	fmt.Printf("BiliGreen %s 已启动：%s\n按 Ctrl+C 退出。\n", version, pageURL)
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "启动失败：", err)
		writeStartupError("BiliGreen 运行失败：" + err.Error())
	}
}

func existingInstance(pageURL string) bool {
	probe := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := probe.Get(pageURL + "/api/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var result struct {
		Name string `json:"name"`
	}
	return resp.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result) == nil && result.Name == "BiliGreen"
}

func writeStartupError(message string) {
	path := filepath.Join(appBaseDir, "BiliGreen-startup-error.txt")
	if err := os.WriteFile(path, []byte(message+"\n"), 0644); err != nil {
		_ = os.WriteFile(filepath.Join(os.TempDir(), "BiliGreen-startup-error.txt"), []byte(message+"\n"), 0644)
	}
}

func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		if !strings.HasPrefix(host, "127.0.0.1:") && !strings.HasPrefix(host, "[::1]:") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct{ URL, Cookie string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		jsonError(w, 400, "请求格式错误")
		return
	}
	bvid, err := extractBVID(in.URL)
	if err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	if in.Cookie == "" {
		in.Cookie = currentCookie()
	}
	var out apiResponse[videoInfo]
	if err := getJSON(r.Context(), "https://api.bilibili.com/x/web-interface/view?bvid="+url.QueryEscape(bvid), in.Cookie, &out); err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	if out.Code != 0 {
		jsonError(w, 502, "B站返回："+out.Message)
		return
	}
	writeJSON(w, out.Data)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct {
		BVID, Title, Part, Cookie, Output, After, Mode string
		CID, Aid, SourceFolder, TargetFolder           int64
		QN, Duration, PageNumber, PageCount            int
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		jsonError(w, 400, "请求格式错误")
		return
	}
	if in.BVID == "" || in.CID == 0 {
		jsonError(w, 400, "缺少视频信息")
		return
	}
	if in.QN == 0 && in.Mode != "audio" {
		in.QN = 80
	}
	if in.Output == "" {
		in.Output = defaultDownloadDir()
	}
	in.Output = expandOutputPath(in.Output)
	if !filepath.IsAbs(in.Output) {
		in.Output = filepath.Join(appBaseDir, in.Output)
	}
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	name := safeName(in.Title)
	if in.Part != "" && in.Part != in.Title {
		name += " - " + safeName(in.Part)
	}
	name += " [" + in.BVID + "]"
	ext := ".mp4"
	if in.Mode == "audio" {
		ext = ".m4a"
	}
	partInfo := ""
	if in.PageCount > 1 {
		partInfo = fmt.Sprintf("P%d/%d · %s", in.PageNumber, in.PageCount, in.Part)
	}
	t := &task{ID: id, Name: name + ext, Status: "排队中", Created: time.Now().UnixNano(), Duration: in.Duration, PartInfo: partInfo, MediaType: in.Mode}
	tasks.Lock()
	tasks.m[id] = t
	tasks.Unlock()
	cookie := in.Cookie
	if cookie == "" {
		cookie = currentCookie()
	}
	go func() {
		downloadSlots <- struct{}{}
		defer func() { <-downloadSlots }()
		setTask(t, func(t *task) { t.Status = "正在获取下载地址" })
		if in.Mode == "audio" {
			if err := downloadAudio(context.Background(), t, in.BVID, in.CID, in.QN, cookie, in.Output); err != nil {
				failTask(t, err)
			}
			if t.Status != "失败" && in.After != "" && in.Aid != 0 && in.SourceFolder != 0 {
				finishFavoriteAction(context.Background(), t, cookie, in.After, in.SourceFolder, in.TargetFolder, in.Aid)
			}
		} else {
			downloadTask(context.Background(), t, in.BVID, in.CID, in.Aid, in.QN, cookie, in.Output, in.After, in.SourceFolder, in.TargetFolder)
		}
	}()
	writeJSON(w, t)
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	tasks.RLock()
	defer tasks.RUnlock()
	list := make([]task, 0, len(tasks.m))
	for _, t := range tasks.m {
		list = append(list, *t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Created > list[j].Created })
	writeJSON(w, list)
}

func downloadTask(ctx context.Context, t *task, bvid string, cid, aid int64, qn int, cookie, output, after string, sourceFolder, targetFolder int64) {
	setTask(t, func(t *task) { t.Status = "正在获取下载地址" })
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=%d&fnval=0&fourk=1", url.QueryEscape(bvid), cid, qn)
	var out apiResponse[playData]
	if err := getJSON(ctx, endpoint, cookie, &out); err != nil {
		failTask(t, err)
		return
	}
	if out.Code != 0 {
		failTask(t, fmt.Errorf("B站返回：%s", out.Message))
		return
	}
	if len(out.Data.DURL) == 0 || out.Data.Quality != qn || qn > 80 {
		if err := downloadDASH(ctx, t, bvid, cid, qn, cookie, output); err != nil {
			failTask(t, err)
		}
		if t.Status == "失败" {
			return
		}
		if after != "" && aid != 0 && sourceFolder != 0 {
			finishFavoriteAction(ctx, t, cookie, after, sourceFolder, targetFolder, aid)
		}
		return
	}
	if len(out.Data.DURL) > 1 {
		failTask(t, errors.New("该视频由多个媒体片段组成，当前版本暂不支持无损拼接"))
		return
	}
	media := out.Data.DURL[0]
	setTask(t, func(t *task) {
		t.Total = media.Size
		t.Status = fmt.Sprintf("下载中（实际清晰度 %d）", out.Data.Quality)
	})
	if err := os.MkdirAll(output, 0755); err != nil {
		failTask(t, err)
		return
	}
	finalPath := filepath.Join(output, t.Name)
	partPath := finalPath + ".part"
	if err := downloadFile(ctx, media.URL, partPath, cookie, t); err != nil {
		failTask(t, err)
		return
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		failTask(t, err)
		return
	}
	setTask(t, func(t *task) { t.Path = finalPath; t.Playable = true })
	if after != "" && aid != 0 && sourceFolder != 0 {
		setTask(t, func(t *task) { t.Status = "下载完成，正在处理收藏夹" })
		if err := applyFavoriteAction(ctx, cookie, after, sourceFolder, targetFolder, aid); err != nil {
			setTask(t, func(t *task) {
				t.Status = "下载完成（收藏夹处理失败）"
				t.Error = err.Error()
			})
			return
		}
		setTask(t, func(t *task) { t.PostAction = "收藏夹处理完成" })
	}
	setTask(t, func(t *task) { t.Status = "完成" })
}

func downloadFile(ctx context.Context, mediaURL, path, cookie string, t *task) error {
	var offset int64
	if st, err := os.Stat(path); err == nil {
		offset = st.Size()
	}
	setTask(t, func(t *task) { t.Downloaded = offset })
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	setHeaders(req, cookie)
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("媒体服务器返回 %s", resp.Status)
	}
	if offset > 0 && resp.StatusCode == http.StatusOK {
		offset = 0
		if err := f.Truncate(0); err != nil {
			return err
		}
		_, _ = f.Seek(0, io.SeekStart)
	}
	buf := make([]byte, 256*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			offset += int64(n)
			now := offset
			setTask(t, func(t *task) { t.Downloaded = now })
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func getJSON(ctx context.Context, endpoint, cookie string, dst any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	setHeaders(req, cookie)
	resp, err := apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("B站接口返回 %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(dst)
}

func setHeaders(req *http.Request, cookie string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 BiliGreen/"+version)
	req.Header.Set("Referer", "https://www.bilibili.com/")
	if strings.TrimSpace(cookie) != "" {
		req.Header.Set("Cookie", strings.TrimSpace(cookie))
	}
}

func extractBVID(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, field := range strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '?' || r == '&' || r == '=' || r == ' ' }) {
		if len(field) >= 12 && strings.EqualFold(field[:2], "BV") {
			return field[:12], nil
		}
	}
	if len(s) == 12 && strings.EqualFold(s[:2], "BV") {
		return s, nil
	}
	return "", errors.New("请输入包含 BV 号的视频链接（暂不支持短链接和 av 号）")
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "bilibili-video"
	}
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	s = r.Replace(s)
	if len([]rune(s)) > 120 {
		s = string([]rune(s)[:120])
	}
	return s
}

func setTask(t *task, fn func(*task)) { tasks.Lock(); defer tasks.Unlock(); fn(t) }
func failTask(t *task, err error) {
	setTask(t, func(t *task) { t.Status = "失败"; t.Error = err.Error() })
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	writeJSON(w, map[string]string{"error": msg})
}

func openBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}
