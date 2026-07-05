package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"xzji/backend/internal/middleware"
	"xzji/backend/internal/models"

	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/webp"
)

// 注：关键词图标上传走前端裁剪后的纯图片字节，不在后端做图像处理；
// 这里只做扩展名 / size / 来源校验、落盘、写 media_files、并把 url 同步到 keyword_responses.icon。

// mediaDir 媒体库存放目录：backend/uploads/media
func mediaDir() string {
	dir := filepath.Join(uploadsDir(), "media")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// iconDir 关键词图标存放目录：backend/uploads/media/icon
func iconDir() string {
	dir := filepath.Join(mediaDir(), "icon")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// allowedImageExts 允许上传的图片扩展名
var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// === 媒体库（媒体库管理页） ===

// AdminMediaList GET /api/admin/media  分页列表，最新在前
func AdminMediaList(c *gin.Context) {
	page := atoiOr(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	limit := atoiOr(c.Query("limit"), 24)
	if limit < 1 {
		limit = 24
	}
	if limit > 100 {
		limit = 100
	}

	var total int64
	db := getApp(c).DB.Model(&models.MediaFile{})
	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		db = db.Where("origin_name LIKE ?", "%"+kw+"%")
	}
	db.Count(&total)

	var files []models.MediaFile
	if err := db.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取媒体列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": files, "total": total, "page": page, "limit": limit})
}

// AdminMediaUpload POST /api/admin/media  multipart 上传，支持多文件字段 file[]
func AdminMediaUpload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		if single, e := c.FormFile("file"); e == nil {
			files = []*multipart.FileHeader{single}
		}
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}

	uid := middleware.GetUserID(c)
	db := getApp(c).DB
	saved := make([]models.MediaFile, 0, len(files))
	for _, fh := range files {
		mf, ferr := saveMediaFile(c, fh, uid, "local")
		if ferr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ferr.Error()})
			return
		}
		if err := db.Create(&mf).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存媒体记录失败"})
			return
		}
		saved = append(saved, mf)
	}
	c.JSON(http.StatusCreated, gin.H{"items": saved})
}

// AdminMediaUploadByURL POST /api/admin/media/url  body: {url}
// 网络图：后端拉取字节后入库落盘，统一进 media_files 管理
func AdminMediaUploadByURL(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供图片 URL"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	parsedURL, err := validateRemoteMediaURL(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const maxBytes = 5 << 20
	resp, err := secureMediaHTTPClient().Get(parsedURL.String())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "拉取网络图片失败"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标图片不可用"})
		return
	}
	body := &io.LimitedReader{R: resp.Body, N: maxBytes + 1}
	data, err := io.ReadAll(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取网络图片失败"})
		return
	}
	if len(data) > maxBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "图片大小不能超过 5MB"})
		return
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标不是有效图片"})
		return
	}
	ext := guessImageExt(resp, req.URL)
	if !allowedImageExts[ext] {
		ext = ".jpg"
	}

	uid := middleware.GetUserID(c)
	filename := fmt.Sprintf("media_%d_%d%s", uid, time.Now().UnixNano(), ext)
	savePath := filepath.Join(mediaDir(), filename)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存图片失败"})
		return
	}
	mime := guessMime(ext)
	w, h := decodeImgDims(savePath)
	mf := models.MediaFile{
		URL:        "/uploads/media/" + filename,
		Filename:   filename,
		OriginName: basenameFromURL(req.URL),
		Mime:       mime,
		Size:       int64(len(data)),
		Width:      w,
		Height:     h,
		Source:     "remote",
		UploaderID: uid,
	}
	if err := getApp(c).DB.Create(&mf).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存媒体记录失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": mf})
}

func validateRemoteMediaURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("URL 格式无效")
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return nil, errors.New("URL 必须以 http(s):// 开头")
	}
	if u.User != nil {
		return nil, errors.New("URL 不允许包含认证信息")
	}
	if isBlockedMediaHost(u.Hostname()) {
		return nil, errors.New("不允许导入内网或本机地址")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	return u, nil
}

func secureMediaHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			var selected net.IP
			for _, ip := range ips {
				if isBlockedMediaIP(ip) {
					return nil, errors.New("不允许连接内网或本机地址")
				}
				if selected == nil && networkAcceptsIP(network, ip) {
					selected = ip
				}
			}
			if selected == nil {
				return nil, errors.New("目标地址不可用")
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(selected.String(), port))
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			_, err := validateRemoteMediaURL(req.URL.String())
			return err
		},
	}
}

func networkAcceptsIP(network string, ip net.IP) bool {
	if strings.HasSuffix(network, "4") {
		return ip.To4() != nil
	}
	if strings.HasSuffix(network, "6") {
		return ip.To4() == nil
	}
	return true
}

func isBlockedMediaHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedMediaIP(ip)
	}
	lower := strings.ToLower(strings.TrimSuffix(host, "."))
	return lower == "localhost" || strings.HasSuffix(lower, ".localhost") || strings.HasSuffix(lower, ".local")
}

func isBlockedMediaIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

// AdminMediaDelete DELETE /api/admin/media/:id  删除文件 + 元数据
func AdminMediaDelete(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}
	var mf models.MediaFile
	if err := getApp(c).DB.First(&mf, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
		return
	}
	// 删除物理文件（忽略不存在）
	if mf.Filename != "" {
		_ = os.Remove(filepath.Join(mediaDir(), mf.Filename))
	}
	if err := getApp(c).DB.Delete(&mf).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// === 关键词图标（裁剪后的字节上传，原图保留在媒体库） ===

// AdminKeywordUploadIcon POST /api/admin/keywords/:id/icon
// 接收 multipart 字段 file（前端裁剪后的方形 PNG/JPEG），落盘到 uploads/media/icon，
// 同步写 keyword_responses.icon，返回新的 icon URL。
// 删除旧的 icon 文件（如果存在）。
func AdminKeywordUploadIcon(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}
	const maxUploadSize = 5 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, err := c.FormFile("file")
	if err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件大小不能超过 5MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的图片"})
		return
	}
	if file.Size > maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件大小不能超过 5MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG、PNG、GIF、WebP 格式的图片"})
		return
	}

	db := getApp(c).DB
	var kw models.KeywordResponse
	if err := db.First(&kw, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "关键词不存在"})
		return
	}

	// 删除旧 icon 文件（仅当指向 media/icon 且与即将保存不同）
	oldIcon := strings.TrimSpace(kw.Icon)
	oldPath := ""
	if oldIcon != "" && strings.HasPrefix(oldIcon, "/uploads/media/icon/") {
		oldPath = filepath.Join(iconDir(), filepath.Base(oldIcon))
	}

	filename := fmt.Sprintf("icon_%d_%d%s", id, time.Now().UnixNano(), ext)
	savePath := filepath.Join(iconDir(), filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图标上传失败"})
		return
	}
	newIcon := "/uploads/media/icon/" + filename

	w, h := decodeImgDims(savePath)
	mf := models.MediaFile{
		URL:        newIcon,
		Filename:   filename,
		OriginName: file.Filename,
		Mime:       guessMime(ext),
		Size:       file.Size,
		Width:      w,
		Height:     h,
		Source:     "local",
		UploaderID: middleware.GetUserID(c),
	}
	if err := db.Create(&mf).Error; err != nil {
		// 元数据写失败不阻塞，仅记录即可
		_ = err
	}

	kw.Icon = newIcon
	if err := db.Save(&kw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新关键词图标失败"})
		return
	}
	// 旧 icon 物理文件清理（落到 keyword_icon 的就是裁后图，不再使用）
	if oldPath != "" {
		_ = os.Remove(oldPath)
	}
	c.JSON(http.StatusOK, gin.H{"icon": newIcon})
}

// AdminKeywordClearIcon DELETE /api/admin/keywords/:id/icon
func AdminKeywordClearIcon(c *gin.Context) {
	id, err := paramID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效ID"})
		return
	}
	db := getApp(c).DB
	var kw models.KeywordResponse
	if err := db.First(&kw, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "关键词不存在"})
		return
	}
	if old := strings.TrimSpace(kw.Icon); old != "" && strings.HasPrefix(old, "/uploads/media/icon/") {
		_ = os.Remove(filepath.Join(iconDir(), filepath.Base(old)))
	}
	kw.Icon = ""
	if err := db.Save(&kw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清除图标失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已清除"})
}

// ============================================

// saveMediaFile 把 multipart 文件落到 mediaDir 并返回元数据（不写 DB）
func saveMediaFile(c *gin.Context, fh *multipart.FileHeader, uid uint, source string) (models.MediaFile, error) {
	const maxUploadSize = 5 << 20
	if fh.Size > maxUploadSize {
		return models.MediaFile{}, errors.New("单文件大小不能超过 5MB")
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedImageExts[ext] {
		return models.MediaFile{}, errors.New("仅支持 JPG、PNG、GIF、WebP 格式的图片")
	}
	filename := fmt.Sprintf("media_%d_%d%s", uid, time.Now().UnixNano(), ext)
	savePath := filepath.Join(mediaDir(), filename)
	if err := c.SaveUploadedFile(fh, savePath); err != nil {
		return models.MediaFile{}, errors.New("保存文件失败")
	}
	w, h := decodeImgDims(savePath)
	return models.MediaFile{
		URL:        "/uploads/media/" + filename,
		Filename:   filename,
		OriginName: fh.Filename,
		Mime:       guessMime(ext),
		Size:       fh.Size,
		Width:      w,
		Height:     h,
		Source:     source,
		UploaderID: uid,
	}, nil
}

// decodeImgDims 解析本地图片宽高，失败返回 0,0
func decodeImgDims(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// guessImageExt 从响应头或 URL 推断扩展名
func guessImageExt(resp *http.Response, url string) string {
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	}
	ext := strings.ToLower(filepath.Ext(url))
	if allowedImageExts[ext] {
		return ext
	}
	return ".jpg"
}

func guessMime(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	}
	return "application/octet-stream"
}

func basenameFromURL(u string) string {
	if i := strings.LastIndex(u, "?"); i > 0 {
		u = u[:i]
	}
	b := filepath.Base(u)
	if b == "" || b == "." || b == "/" {
		return "remote_image"
	}
	return b
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
