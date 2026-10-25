// Package service 把原 HTTP handler 的能力迁移为可被前端直接调用的
// mygo service：方法签名即 IPC 接口，事件替代轮询。
package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"course-grabber/internal/auth"
	"course-grabber/internal/grabber"

	"github.com/egoist/mygo"
)

// GrabEvent 抢课事件流：结果、状态变化、会话过期都从这里推给前端
type GrabEvent struct {
	Kind       string `json:"kind"` // result | running | finished
	CourseID   string `json:"courseId,omitempty"`
	CourseName string `json:"courseName,omitempty"`
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	Time       string `json:"time,omitempty"`
}

var GrabUpdated = mygo.NewEvent[GrabEvent]("grabUpdated")

type SessionEvent struct {
	LoggedIn bool `json:"loggedIn"`
	Expired  bool `json:"expired"` // true 表示 JSESSIONID 失效而非主动登出
}

var SessionChanged = mygo.NewEvent[SessionEvent]("sessionChanged")

// SemesterInfo 从选课页面解析出的学期状态
type SemesterInfo struct {
	Semester   string `json:"semester"`
	CourseType string `json:"courseType"`
	Stage      string `json:"stage"`
	Limit      int    `json:"limit"`
	CanDrop    bool   `json:"canDrop"`
}

// Service App 的全部后端能力，以 "App" 名字绑定给前端
type Service struct {
	mu      sync.Mutex
	auth    *auth.GDUTAuth
	grabber *grabber.Grabber

	version string
}

func New(version string) *Service {
	return &Service{
		auth:    auth.NewGDUTAuth(),
		version: version,
	}
}

// ---------- 会话 ----------

type SessionStatus struct {
	LoggedIn bool   `json:"loggedIn"`
	Version  string `json:"version"`
}

// Status 当前登录状态与应用版本
func (s *Service) Status() SessionStatus {
	return SessionStatus{LoggedIn: s.auth.IsLoggedIn(), Version: s.version}
}

// ImportSession 导入 JSESSIONID 建立会话
func (s *Service) ImportSession(jsessionid string) error {
	jsessionid = strings.TrimSpace(jsessionid)
	if jsessionid == "" {
		return errors.New("请填写 JSESSIONID")
	}

	s.mu.Lock()
	s.auth.SetJSESSIONID(jsessionid)
	s.grabber = grabber.New(s.auth.GetClient())
	s.grabber.SetOnResult(func(r grabber.GrabResult) {
		GrabUpdated.Broadcast(GrabEvent{
			Kind:       "result",
			CourseID:   r.CourseID,
			CourseName: r.CourseName,
			Success:    r.Success,
			Message:    r.Message,
			Time:       r.Time.Format("15:04:05"),
		})
		if r.Success {
			notifySuccess(r.CourseName)
		}
	})
	s.grabber.SetOnStop(func(reason string) {
		msg := "抢课已停止"
		if reason == "completed" {
			msg = "清单课程已全部处理完"
		}
		GrabUpdated.Broadcast(GrabEvent{Kind: "finished", Message: msg})
	})
	s.mu.Unlock()

	SessionChanged.Broadcast(SessionEvent{LoggedIn: true})
	return nil
}

// Logout 登出并停止抢课
func (s *Service) Logout() {
	s.mu.Lock()
	if s.grabber != nil {
		s.grabber.StopGrabbing()
	}
	s.auth.Logout()
	s.grabber = nil
	s.mu.Unlock()

	SessionChanged.Broadcast(SessionEvent{LoggedIn: false})
}

// ---------- 课程 ----------

// ListCourses 拉取全部可选课程（自动翻页、去重）
func (s *Service) ListCourses() ([]grabber.Course, error) {
	g, err := s.currentGrabber()
	if err != nil {
		return nil, err
	}

	var all []grabber.Course
	total := 0
	rows := 100
	for page := 1; page <= 30; page++ {
		cp, err := g.FetchCourses(page, rows)
		if err != nil {
			return nil, s.wrapSessionError(err)
		}
		if page == 1 {
			total = cp.Total
		}
		all = append(all, cp.Courses...)
		if len(cp.Courses) < rows || len(all) >= total {
			break
		}
	}

	seen := make(map[string]bool, len(all))
	deduped := all[:0]
	for _, c := range all {
		if c.ID == "" || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		deduped = append(deduped, c)
	}
	return deduped, nil
}

// SelectedCourses 已选课程
func (s *Service) SelectedCourses() ([]grabber.Course, error) {
	g, err := s.currentGrabber()
	if err != nil {
		return nil, err
	}
	courses, err := g.FetchSelectedCourses()
	if err != nil {
		return nil, s.wrapSessionError(err)
	}
	return courses, nil
}

// DropCourse 退选课程，返回结果对象（Success=false 也是正常返回，前端据此展示）
func (s *Service) DropCourse(courseID, courseName, jxbdm string) grabber.GrabResult {
	g, err := s.currentGrabber()
	if err != nil {
		return grabber.GrabResult{CourseID: courseID, CourseName: courseName, Success: false, Message: err.Error(), Time: time.Now()}
	}
	return g.DropCourse(courseID, courseName, jxbdm)
}

// CourseSchedule 单门课程的上课时间
func (s *Service) CourseSchedule(kcrwdm string) ([]grabber.CourseSchedule, error) {
	if kcrwdm == "" {
		return nil, errors.New("请提供课程ID")
	}
	g, err := s.currentGrabber()
	if err != nil {
		return nil, err
	}
	schedules, err := g.FetchCourseSchedule(kcrwdm)
	if err != nil {
		return nil, s.wrapSessionError(err)
	}
	return schedules, nil
}

var (
	reSemester = regexp.MustCompile(`选课学期:([^,]+)`)
	reType     = regexp.MustCompile(`,([^,]+),现在是`)
	reStage    = regexp.MustCompile(`现在是(.+?)时间`)
	reLimit    = regexp.MustCompile(`限选\s*(\d+)`)
	reLimitLoose = regexp.MustCompile(`限选.*?(\d+).*?门`)
)

// SemesterInfo 解析选课页面信息（学期、阶段、限选门数、能否退选）
func (s *Service) SemesterInfo() (*SemesterInfo, error) {
	g, err := s.currentGrabber()
	if err != nil {
		return nil, err
	}
	html, err := g.FetchPageInfo()
	if err != nil {
		return nil, s.wrapSessionError(err)
	}

	info := &SemesterInfo{Semester: "未知", CourseType: "未知", Stage: "未知"}
	if m := reSemester.FindStringSubmatch(html); m != nil {
		info.Semester = strings.TrimSpace(m[1])
	}
	if m := reType.FindStringSubmatch(html); m != nil {
		info.CourseType = strings.TrimSpace(m[1])
	}
	if m := reStage.FindStringSubmatch(html); m != nil {
		info.Stage = strings.TrimSpace(m[1])
	}
	if m := reLimit.FindStringSubmatch(html); m != nil {
		info.Limit = atoi(m[1])
	} else if m := reLimitLoose.FindStringSubmatch(html); m != nil {
		info.Limit = atoi(m[1])
	}
	info.CanDrop = strings.Contains(html, "可退选")
	return info, nil
}

// ---------- 抢课 ----------

// StartGrab 按愿望清单顺序并发抢课
func (s *Service) StartGrab(courseIDs []string, courseNames map[string]string, intervalSec int) error {
	if len(courseIDs) == 0 {
		return errors.New("愿望清单为空")
	}
	g, err := s.currentGrabber()
	if err != nil {
		return err
	}
	if g.IsRunning() {
		return errors.New("抢课已在进行中")
	}
	if intervalSec < 1 {
		intervalSec = 1
	}
	if courseNames == nil {
		courseNames = map[string]string{}
	}
	g.StartGrabbing(courseIDs, courseNames, time.Duration(intervalSec)*time.Second)
	GrabUpdated.Broadcast(GrabEvent{Kind: "running"})
	return nil
}

// StopGrab 停止抢课
func (s *Service) StopGrab() {
	s.mu.Lock()
	g := s.grabber
	s.mu.Unlock()
	if g != nil {
		g.StopGrabbing()
	}
}

// GrabRunning 是否正在抢课
func (s *Service) GrabRunning() bool {
	s.mu.Lock()
	g := s.grabber
	s.mu.Unlock()
	return g != nil && g.IsRunning()
}

// GrabResults 抢课结果历史
func (s *Service) GrabResults() []grabber.GrabResult {
	s.mu.Lock()
	g := s.grabber
	s.mu.Unlock()
	if g == nil {
		return []grabber.GrabResult{}
	}
	return g.GetResults()
}

// ClearGrabResults 清空结果历史
func (s *Service) ClearGrabResults() {
	s.mu.Lock()
	g := s.grabber
	s.mu.Unlock()
	if g != nil {
		g.ClearResults()
	}
}

// ---------- 内部 ----------

func (s *Service) currentGrabber() (*grabber.Grabber, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.auth.IsLoggedIn() {
		return nil, errors.New("请先导入 JSESSIONID 登录")
	}
	if s.grabber == nil {
		return nil, errors.New("抢课器未初始化")
	}
	return s.grabber, nil
}

// wrapSessionError 把会话失效的报错转成主动登出 + 事件广播
func (s *Service) wrapSessionError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "未登录") || strings.Contains(msg, "过期") || strings.Contains(msg, "session") {
		s.Logout()
		SessionChanged.Broadcast(SessionEvent{LoggedIn: false, Expired: true})
		return fmt.Errorf("登录已失效，请重新导入 JSESSIONID")
	}
	return err
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// notifySuccess 抢课成功时弹系统通知（窗口隐藏时最有用）
func notifySuccess(courseName string) {
	if !mygo.NotificationsSupported() {
		return
	}
	n := mygo.NewNotification(mygo.NotificationOptions{
		Title: "抢课成功",
		Body:  courseName,
	})
	_ = n.Show()
}
