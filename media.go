package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type qualityOption struct {
	ID     int    `json:"id"`
	Label  string `json:"label"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	DASH   bool   `json:"dash"`
}

type audioQualityOption struct {
	ID      int    `json:"id"`
	Label   string `json:"label"`
	Bitrate int64  `json:"bitrate"`
	Codec   string `json:"codec,omitempty"`
}

func fetchDASH(ctx context.Context, bvid string, cid int64, cookie string) (*playData, error) {
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=127&fnval=4048&fourk=1", url.QueryEscape(bvid), cid)
	var out apiResponse[playData]
	if err := getJSON(ctx, endpoint, cookie, &out); err != nil {
		return nil, err
	}
	if out.Code != 0 || out.Data.Dash == nil {
		return nil, fmt.Errorf("B站未返回 DASH 媒体流：%s", out.Message)
	}
	return &out.Data, nil
}

func handleAudioQualities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct {
		BVID string
		CID  int64
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil || in.BVID == "" || in.CID == 0 {
		jsonError(w, 400, "缺少视频信息")
		return
	}
	data, err := fetchDASH(r.Context(), in.BVID, in.CID, currentCookie())
	if err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	seen := map[string]bool{}
	list := make([]audioQualityOption, 0, len(data.Dash.Audio))
	for _, stream := range data.Dash.Audio {
		key := fmt.Sprintf("%d/%s", (stream.Bandwidth+500)/1000, stream.Codecs)
		if seen[key] {
			continue
		}
		seen[key] = true
		kbps := (stream.Bandwidth + 500) / 1000
		list = append(list, audioQualityOption{ID: stream.ID, Label: fmt.Sprintf("%d kbps", kbps), Bitrate: stream.Bandwidth, Codec: stream.Codecs})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Bitrate > list[j].Bitrate })
	writeJSON(w, list)
}

func downloadAudio(ctx context.Context, t *task, bvid string, cid int64, audioID int, cookie, output string) error {
	data, err := fetchDASH(ctx, bvid, cid, cookie)
	if err != nil {
		return err
	}
	var selected dashStream
	for _, stream := range data.Dash.Audio {
		if (audioID == 0 || stream.ID == audioID) && stream.Bandwidth > selected.Bandwidth {
			selected = stream
		}
	}
	if selected.mediaURL() == "" {
		return errors.New("当前视频没有所选音质的音频流")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	finalPath := filepath.Join(output, t.Name)
	partPath := finalPath + ".part"
	setTask(t, func(t *task) { t.Status = fmt.Sprintf("下载音频（%d kbps）", (selected.Bandwidth+500)/1000) })
	if err := downloadFile(ctx, selected.mediaURL(), partPath, cookie, t); err != nil {
		return err
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		return err
	}
	finalSize := t.Downloaded
	if info, err := os.Stat(finalPath); err == nil {
		finalSize = info.Size()
	}
	setTask(t, func(t *task) {
		t.Downloaded = finalSize
		t.Total = finalSize
		t.Status = fmt.Sprintf("完成（音频 %d kbps）", (selected.Bandwidth+500)/1000)
		t.Path = finalPath
		t.Playable = true
	})
	return nil
}

func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(appBaseDir, "downloads")
	}
	return filepath.Join(home, "Downloads", "BiliGreen")
}

func expandOutputPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimLeft(path[1:], `/\`))
		}
	}
	return filepath.Clean(path)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"defaultDownloadDir": defaultDownloadDir(), "appDir": appBaseDir})
}

func handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct{ Path string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		jsonError(w, 400, "目录格式错误")
		return
	}
	path := expandOutputPath(in.Path)
	if path == "." || path == "" {
		path = defaultDownloadDir()
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(appBaseDir, path)
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		jsonError(w, 500, "无法打开目录："+err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": path})
}

func handleQualities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var in struct {
		BVID string
		CID  int64
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil || in.BVID == "" || in.CID == 0 {
		jsonError(w, 400, "缺少视频信息")
		return
	}
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=127&fnval=4048&fourk=1", url.QueryEscape(in.BVID), in.CID)
	var out apiResponse[playData]
	if err := getJSON(r.Context(), endpoint, currentCookie(), &out); err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	if out.Code != 0 {
		jsonError(w, 502, "B站返回："+out.Message)
		return
	}
	labels := map[int]string{}
	for _, f := range out.Data.SupportFormats {
		label := f.NewDescription
		if label == "" {
			label = f.DisplayDesc
		}
		labels[f.Quality] = label
	}
	options := map[int]qualityOption{}
	if out.Data.Dash != nil {
		for _, stream := range out.Data.Dash.Video {
			label := labels[stream.ID]
			if label == "" {
				label = qualityLabel(stream.ID)
			}
			options[stream.ID] = qualityOption{ID: stream.ID, Label: label, Width: stream.Width, Height: stream.Height, DASH: true}
		}
	}
	for _, id := range out.Data.AcceptQuality {
		if _, ok := options[id]; !ok && id == out.Data.Quality && len(out.Data.DURL) > 0 {
			label := labels[id]
			if label == "" {
				label = qualityLabel(id)
			}
			options[id] = qualityOption{ID: id, Label: label}
		}
	}
	list := make([]qualityOption, 0, len(options))
	for _, option := range options {
		list = append(list, option)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID > list[j].ID })
	writeJSON(w, list)
}

func qualityLabel(id int) string {
	labels := map[int]string{127: "8K", 126: "杜比视界", 125: "HDR", 120: "4K", 116: "1080P 60帧", 112: "1080P+", 80: "1080P", 74: "720P 60帧", 64: "720P", 32: "480P", 16: "360P"}
	if label := labels[id]; label != "" {
		return label
	}
	return "清晰度 " + strconv.Itoa(id)
}

func downloadDASH(ctx context.Context, t *task, bvid string, cid int64, qn int, cookie, output string) error {
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=127&fnval=4048&fourk=1", url.QueryEscape(bvid), cid)
	var out apiResponse[playData]
	if err := getJSON(ctx, endpoint, cookie, &out); err != nil {
		return err
	}
	if out.Code != 0 || out.Data.Dash == nil {
		return fmt.Errorf("B站未返回 DASH 高画质流：%s", out.Message)
	}
	requested := qn
	if requested == 127 {
		requested = 0
		for _, s := range out.Data.Dash.Video {
			if s.ID > requested {
				requested = s.ID
			}
		}
	}
	var video dashStream
	for _, s := range out.Data.Dash.Video {
		if s.ID == requested && (video.mediaURL() == "" || s.Codecid == 7) {
			video = s
		}
	}
	var audio dashStream
	for _, s := range out.Data.Dash.Audio {
		if s.Bandwidth > audio.Bandwidth {
			audio = s
		}
	}
	if video.mediaURL() == "" {
		return fmt.Errorf("当前账号没有 %s 可用视频流", qualityLabel(requested))
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	finalPath := filepath.Join(output, t.Name)
	videoPath, audioPath, muxPath := finalPath+".video.m4s", finalPath+".audio.m4s", finalPath+".mux.part.mp4"
	setTask(t, func(t *task) {
		t.Total = 0
		t.Downloaded = 0
		t.Status = "下载视频轨（" + qualityLabel(requested) + "）"
	})
	if err := downloadFile(ctx, video.mediaURL(), videoPath, cookie, t); err != nil {
		return err
	}
	if audio.mediaURL() == "" {
		if err := os.Rename(videoPath, finalPath); err != nil {
			return err
		}
		finalSize := t.Downloaded
		if info, err := os.Stat(finalPath); err == nil {
			finalSize = info.Size()
		}
		setTask(t, func(t *task) {
			t.Downloaded = finalSize
			t.Total = finalSize
			t.Status = "完成（" + qualityLabel(requested) + " · 原视频无音轨）"
			t.Path = finalPath
			t.Playable = true
		})
		return nil
	}
	setTask(t, func(t *task) { t.Total = 0; t.Downloaded = 0; t.Status = "下载音频轨" })
	if err := downloadFile(ctx, audio.mediaURL(), audioPath, cookie, t); err != nil {
		return err
	}
	var combinedSize int64
	if info, err := os.Stat(videoPath); err == nil {
		combinedSize += info.Size()
	}
	if info, err := os.Stat(audioPath); err == nil {
		combinedSize += info.Size()
	}
	setTask(t, func(t *task) { t.Downloaded = combinedSize; t.Total = combinedSize })
	setTask(t, func(t *task) { t.Status = "正在无损合并音视频" })
	ffmpeg, ffmpegErr := findFFmpeg()
	if ffmpegErr == nil && os.Getenv("BILIGREEN_PURE_GO_MUX") == "" {
		cmd := exec.CommandContext(ctx, ffmpeg, "-y", "-loglevel", "error", "-i", videoPath, "-i", audioPath, "-c", "copy", muxPath)
		if outputBytes, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("音视频合并失败：%v %s", err, strings.TrimSpace(string(outputBytes)))
		}
	} else if err := mergeM4S(videoPath, audioPath, muxPath); err != nil {
		return fmt.Errorf("内置合并器失败：%w", err)
	}
	if err := os.Rename(muxPath, finalPath); err != nil {
		return err
	}
	_ = os.Remove(videoPath)
	_ = os.Remove(audioPath)
	setTask(t, func(t *task) {
		t.Status = "完成（" + qualityLabel(requested) + "）"
		t.Path = finalPath
		t.Playable = true
	})
	return nil
}

func handleMedia(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	tasks.RLock()
	t, ok := tasks.m[id]
	path := ""
	if ok && t.Playable {
		path = t.Path
	}
	tasks.RUnlock()
	if path == "" {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	if strings.EqualFold(filepath.Ext(info.Name()), ".m4a") {
		w.Header().Set("Content-Type", "audio/mp4")
	} else {
		w.Header().Set("Content-Type", "video/mp4")
	}
	w.Header().Set("Content-Disposition", "inline")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func findFFmpeg() (string, error) {
	name := "ffmpeg"
	if filepath.Ext(os.Args[0]) == ".exe" {
		name = "ffmpeg.exe"
	}
	sibling := filepath.Join(appBaseDir, name)
	if st, err := os.Stat(sibling); err == nil && !st.IsDir() {
		return sibling, nil
	}
	if found, err := exec.LookPath("ffmpeg"); err == nil {
		return found, nil
	}
	return "", errors.New("未找到外部合并器")
}

func finishFavoriteAction(ctx context.Context, t *task, cookie, after string, sourceFolder, targetFolder, aid int64) {
	setTask(t, func(t *task) { t.Status = "下载完成，正在处理收藏夹" })
	if err := applyFavoriteAction(ctx, cookie, after, sourceFolder, targetFolder, aid); err != nil {
		setTask(t, func(t *task) { t.Status = "下载完成（收藏夹处理失败）"; t.Error = err.Error() })
		return
	}
	setTask(t, func(t *task) { t.Status = "完成"; t.PostAction = "收藏夹处理完成" })
}
