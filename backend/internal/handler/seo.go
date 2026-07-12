package handler

import (
	"encoding/xml"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
	"time"
	"xzji/backend/internal/models"
)

var publicSEOKeys = []string{"siteName", "siteDescription", "siteLogoUrl", "seoEnabled", "seoSiteUrl", "seoTitleTemplate", "seoDefaultKeywords", "seoDefaultImage", "seoRobotsIndex", "seoGoogleVerification", "seoBingVerification", "geoEnabled", "geoOrganizationName", "geoOrganizationDescription", "geoOrganizationLogo", "geoAllowAICrawlers", "geoLlmsDescription", "geoContactEmail", "geoSocialProfiles"}

func loadSEOSettings(c *gin.Context) map[string]string {
	var rows []models.SiteSetting
	getApp(c).DB.Where("setting_key IN ?", publicSEOKeys).Find(&rows)
	out := map[string]string{"seoEnabled": "true", "seoRobotsIndex": "true", "geoEnabled": "true", "geoAllowAICrawlers": "true"}
	for _, r := range rows {
		out[r.SettingKey] = r.SettingValue
	}
	return out
}
func PublicSEOSettings(c *gin.Context) { c.JSON(http.StatusOK, loadSEOSettings(c)) }
func canonicalBase(c *gin.Context, s map[string]string) string {
	base := strings.TrimRight(strings.TrimSpace(s["seoSiteUrl"]), "/")
	if base != "" {
		return base
	}
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}
func RobotsTxt(c *gin.Context) {
	s := loadSEOSettings(c)
	var text string
	if s["seoEnabled"] != "false" && s["seoRobotsIndex"] != "false" {
		text = "User-agent: *\nAllow: /\nDisallow: /admin/\nDisallow: /user/\nDisallow: /api/\n"
	} else {
		text = "User-agent: *\nDisallow: /\n"
	}
	if s["geoAllowAICrawlers"] == "false" {
		for _, bot := range []string{"GPTBot", "ChatGPT-User", "ClaudeBot", "Google-Extended", "PerplexityBot", "CCBot"} {
			text += fmt.Sprintf("\nUser-agent: %s\nDisallow: /\n", bot)
		}
	}
	text += "\nSitemap: " + canonicalBase(c, s) + "/sitemap.xml\n"
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(text))
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}
type sitemapSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func SitemapXML(c *gin.Context) {
	s := loadSEOSettings(c)
	if s["seoEnabled"] == "false" || s["seoRobotsIndex"] == "false" {
		c.Status(http.StatusNotFound)
		return
	}
	base := canonicalBase(c, s)
	urls := []sitemapURL{{Loc: base + "/", ChangeFreq: "daily", Priority: "1.0"}, {Loc: base + "/categories", ChangeFreq: "weekly", Priority: "0.8"}}
	var pages []models.Page
	getApp(c).DB.Where("is_published = ?", true).Order("updated_at DESC").Find(&pages)
	for _, p := range pages {
		urls = append(urls, sitemapURL{Loc: base + "/page/" + url.PathEscape(p.Slug), LastMod: p.UpdatedAt.UTC().Format("2006-01-02"), ChangeFreq: "monthly", Priority: "0.6"})
	}
	payload, _ := xml.MarshalIndent(sitemapSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}, "", "  ")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", append([]byte(xml.Header), payload...))
}
func LlmsTxt(c *gin.Context) {
	s := loadSEOSettings(c)
	if s["geoEnabled"] == "false" {
		c.Status(http.StatusNotFound)
		return
	}
	name := s["geoOrganizationName"]
	if name == "" {
		name = s["siteName"]
	}
	if name == "" {
		name = "XZJI"
	}
	desc := s["geoLlmsDescription"]
	if desc == "" {
		desc = s["geoOrganizationDescription"]
	}
	if desc == "" {
		desc = s["siteDescription"]
	}
	base := canonicalBase(c, s)
	text := fmt.Sprintf("# %s\n\n> %s\n\n## 核心页面\n\n- [首页](%s/): 站点主要内容与关键词检索入口\n- [分类](%s/categories): 按主题浏览内容\n", name, desc, base, base)
	var pages []models.Page
	getApp(c).DB.Where("is_published = ?", true).Order("updated_at DESC").Limit(100).Find(&pages)
	if len(pages) > 0 {
		text += "\n## 公开内容\n\n"
		for _, p := range pages {
			text += fmt.Sprintf("- [%s](%s/page/%s)\n", p.Title, base, url.PathEscape(p.Slug))
		}
	}
	text += fmt.Sprintf("\n生成时间：%s\n", time.Now().UTC().Format(time.RFC3339))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(text))
}
