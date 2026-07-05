package auth

import (
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "time"
)

const (
    GDUTBaseURL = "https://jxfw.gdut.edu.cn"
)

// GDUTAuth GDUT教务系统认证
type GDUTAuth struct {
    client     *http.Client
    jsessionid string
    loggedIn   bool
}

// NewGDUTAuth 创建GDUT认证实例
func NewGDUTAuth() *GDUTAuth {
    return &GDUTAuth{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        loggedIn: false,
    }
}

// SetJSESSIONID 设置JSESSIONID
func (g *GDUTAuth) SetJSESSIONID(jsessionid string) {
    g.jsessionid = jsessionid

    // 创建cookie jar并设置JSESSIONID
    jar, _ := cookiejar.New(nil)
    u, _ := url.Parse(GDUTBaseURL)
    jar.SetCookies(u, []*http.Cookie{
        {
            Name:  "JSESSIONID",
            Value: jsessionid,
        },
    })
    g.client.Jar = jar
    g.loggedIn = true
}

// IsLoggedIn 是否已登录
func (g *GDUTAuth) IsLoggedIn() bool {
    return g.loggedIn
}

// GetClient 获取HTTP客户端
func (g *GDUTAuth) GetClient() *http.Client {
    return g.client
}

// Logout 登出
func (g *GDUTAuth) Logout() {
    g.loggedIn = false
    g.jsessionid = ""
    g.client.Jar = nil
}
