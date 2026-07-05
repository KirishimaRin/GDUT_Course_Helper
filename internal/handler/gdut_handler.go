package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"course-grabber/internal/auth"
	"course-grabber/internal/grabber"

	"github.com/gin-gonic/gin"
)

// GDUTHandler GDUT教务系统处理器
type GDUTHandler struct {
	auth    *auth.GDUTAuth
	grabber *grabber.Grabber
}

// NewGDUTHandler 创建GDUT处理器
func NewGDUTHandler() *GDUTHandler {
	return &GDUTHandler{
		auth: auth.NewGDUTAuth(),
	}
}

// SetJSESSIONID 设置JSESSIONID
func (h *GDUTHandler) SetJSESSIONID(c *gin.Context) {
	var req struct {
		JSESSIONID string `json:"jsessionid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供JSESSIONID"})
		return
	}

	h.auth.SetJSESSIONID(req.JSESSIONID)
	h.grabber = grabber.New(h.auth.GetClient())

	c.JSON(http.StatusOK, gin.H{
		"message":   "导入成功",
		"logged_in": true,
	})
}

// GetStatus 获取登录状态
func (h *GDUTHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"logged_in": h.auth.IsLoggedIn(),
	})
}

// GetPageInfo 获取页面信息（调试用）
func (h *GDUTHandler) GetPageInfo(c *gin.Context) {
	if !h.auth.IsLoggedIn() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	if h.grabber == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
		return
	}

	html, err := h.grabber.FetchPageInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"html": html})
}

// GetCourseSchedule 获取课程上课时间
func (h *GDUTHandler) GetCourseSchedule(c *gin.Context) {
    if !h.auth.IsLoggedIn() {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
        return
    }

    if h.grabber == nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
        return
    }

    kcrwdm := c.Query("kcrwdm")
    if kcrwdm == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请提供课程ID"})
        return
    }

    schedules, err := h.grabber.FetchCourseSchedule(kcrwdm)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// GetAllSchedules 获取学生所有已选课程的上课时间
func (h *GDUTHandler) GetAllSchedules(c *gin.Context) {
    if !h.auth.IsLoggedIn() {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
        return
    }

    if h.grabber == nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
        return
    }

    schedules, err := h.grabber.FetchAllCourseSchedules()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// GetSelectedCourses 获取已选课程
func (h *GDUTHandler) GetSelectedCourses(c *gin.Context) {
	if !h.auth.IsLoggedIn() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	if h.grabber == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
		return
	}

	courses, err := h.grabber.FetchSelectedCourses()
	if err != nil {
		if strings.Contains(err.Error(), "未登录") || strings.Contains(err.Error(), "过期") || strings.Contains(err.Error(), "session") {
			h.auth.Logout()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"courses": courses, "total": len(courses)})
}

// GetCourses 获取课程列表（分页）
func (h *GDUTHandler) GetCourses(c *gin.Context) {
	if !h.auth.IsLoggedIn() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	if h.grabber == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	rows, _ := strconv.Atoi(c.DefaultQuery("rows", "20"))

	if page < 1 {
		page = 1
	}
	if rows < 1 || rows > 100 {
		rows = 20
	}

	fmt.Printf("[%s] Fetching courses page=%d rows=%d\n", time.Now().Format("15:04:05"), page, rows)

	result, err := h.grabber.FetchCourses(page, rows)
	if err != nil {
		fmt.Printf("[%s] Error: %v\n", time.Now().Format("15:04:05"), err)
		if strings.Contains(err.Error(), "未登录") || strings.Contains(err.Error(), "过期") || strings.Contains(err.Error(), "session") {
			h.auth.Logout()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[%s] Found %d courses (page %d/%d)\n", time.Now().Format("15:04:05"), len(result.Courses), result.Page, (result.Total+result.Rows-1)/result.Rows)
	c.JSON(http.StatusOK, result)
}

// StartGrabbing 开始抢课
func (h *GDUTHandler) StartGrabbing(c *gin.Context) {
	if !h.auth.IsLoggedIn() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	if h.grabber == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
		return
	}

	var req struct {
		CourseIDs   []string          `json:"course_ids" binding:"required"`
		CourseNames map[string]string `json:"course_names"`
		Interval    int               `json:"interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供要抢的课程ID"})
		return
	}

	interval := time.Duration(req.Interval) * time.Second
	if interval < 1*time.Second {
		interval = 1 * time.Second
	}

	h.grabber.StartGrabbing(req.CourseIDs, req.CourseNames, interval)

	c.JSON(http.StatusOK, gin.H{"message": "开始抢课"})
}

// StopGrabbing 停止抢课
func (h *GDUTHandler) StopGrabbing(c *gin.Context) {
	if h.grabber == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
		return
	}

	h.grabber.StopGrabbing()

	c.JSON(http.StatusOK, gin.H{"message": "停止抢课"})
}

// GetGrabStatus 获取抢课状态
func (h *GDUTHandler) GetGrabStatus(c *gin.Context) {
	if h.grabber == nil {
		c.JSON(http.StatusOK, gin.H{
			"running": false,
			"results": []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"running": h.grabber.IsRunning(),
		"results": h.grabber.GetResults(),
	})
}

// DropCourse 退选课程
func (h *GDUTHandler) DropCourse(c *gin.Context) {
	if !h.auth.IsLoggedIn() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	if h.grabber == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "抢课器未初始化"})
		return
	}

	var req struct {
		CourseID   string `json:"course_id" binding:"required"`
		CourseName string `json:"course_name"`
		Jxbdm      string `json:"jxbdm"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供课程ID"})
		return
	}

	result := h.grabber.DropCourse(req.CourseID, req.CourseName, req.Jxbdm)

	if result.Success {
		c.JSON(http.StatusOK, gin.H{"message": result.Message})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
	}
}

// ClearResults 清空抢课结果
func (h *GDUTHandler) ClearResults(c *gin.Context) {
	if h.grabber != nil {
		h.grabber.ClearResults()
	}
	c.JSON(http.StatusOK, gin.H{"message": "已清空"})
}

// Logout 登出
func (h *GDUTHandler) Logout(c *gin.Context) {
	h.auth.Logout()
	if h.grabber != nil {
		h.grabber.StopGrabbing()
	}
	c.JSON(http.StatusOK, gin.H{"message": "已登出"})
}
