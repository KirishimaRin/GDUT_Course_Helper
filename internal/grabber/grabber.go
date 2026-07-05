package grabber

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	GDUTBaseURL       = "https://jxfw.gdut.edu.cn"
	CourseListAPI     = "/xsxklist!getDataList.action"
	SelectCourseAPI   = "/xsxklist!getAdd.action"
	SelectedCourseAPI = "/xsxklist!getXzkcList.action"
	DropCourseAPI     = "/xsxklist!getCancel.action"
	PageInfoAPI       = "/xsxklist!xsmhxsxk.action"
)

// Course 课程信息
type Course struct {
	ID       string `json:"kcrwdm"`
	Code     string `json:"kcdm"`
	Name     string `json:"kcmc"`
	Teacher  string `json:"teaxm"`
	Category string `json:"kcdlmc"`
	Type     string `json:"kcflmc"`
	Credits  string `json:"xf"`
	Hours    string `json:"zxs"`
	Capacity int    `json:"pkrs"`
	Enrolled int    `json:"jxbrs"`
	Jxbdm    string `json:"jxbdm"`
}

func (c Course) GetAvailable() int {
	return c.Capacity - c.Enrolled
}

func (c Course) IsFull() bool {
	return c.Enrolled >= c.Capacity
}

// CoursePage 课程分页结果
type CoursePage struct {
	Courses []Course `json:"courses"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	Rows    int      `json:"rows"`
}

// GrabResult 抢课结果
type GrabResult struct {
	CourseID   string    `json:"course_id"`
	CourseName string    `json:"course_name"`
	Success    bool      `json:"success"`
	Message    string    `json:"message"`
	Time       time.Time `json:"time"`
}

// Grabber 抢课器
type Grabber struct {
	client   *http.Client
	results  []GrabResult
	running  bool
	mutex    sync.Mutex
	stopChan chan bool
}

func New(client *http.Client) *Grabber {
	return &Grabber{
		client:   client,
		stopChan: make(chan bool),
	}
}

// FetchCourses 获取课程列表（分页）
func (g *Grabber) FetchCourses(page, rows int) (*CoursePage, error) {
	if page < 1 {
		page = 1
	}
	if rows < 1 {
		rows = 20
	}

	data := fmt.Sprintf("sort=kcrwdm&order=asc&page=%d&rows=%d", page, rows)

	req, err := http.NewRequest("POST", GDUTBaseURL+CourseListAPI, strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", GDUTBaseURL+"/xsxklist!xsmhxsxk.action")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	bodyStr := string(body)
	if strings.HasPrefix(strings.TrimSpace(bodyStr), "<!DOCTYPE") {
		return nil, fmt.Errorf("未登录或session已过期，请重新导入JSESSIONID")
	}

	var result struct {
		Rows  []json.RawMessage `json:"rows"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	courses, err := parseCoursesFromRaw(result.Rows)
	if err != nil {
		return nil, err
	}

	return &CoursePage{
		Courses: courses,
		Total:   result.Total,
		Page:    page,
		Rows:    rows,
	}, nil
}

// FetchPageInfo 获取页面信息（调试用）
func (g *Grabber) FetchPageInfo() (string, error) {
	req, err := http.NewRequest("GET", GDUTBaseURL+PageInfoAPI, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", GDUTBaseURL+"/xsxklist!xsmhxsxk.action")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	return string(body), nil
}

// CourseSchedule 课程上课时间
type CourseSchedule struct {
    Kcrwdm string `json:"kcrwdm"` // 课程ID
    Kcmc   string `json:"kcmc"`   // 课程名称
    Zc     string `json:"zc"`     // 周次
    Xq     string `json:"xq"`     // 星期
    Jcdm2  string `json:"jcdm2"`  // 节次代码
}

// FetchCourseSchedule 获取课程上课时间
func (g *Grabber) FetchCourseSchedule(kcrwdm string) ([]CourseSchedule, error) {
    data := fmt.Sprintf("kcrwdms=%s", kcrwdm)

    req, err := http.NewRequest("GET", GDUTBaseURL+"/xsxklist!getSksj.action?"+data, nil)
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %v", err)
    }

    req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    req.Header.Set("Referer", GDUTBaseURL+"/xsxklist!xsmhxsxk.action")

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("请求失败: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %v", err)
    }

    bodyStr := string(body)
    if strings.HasPrefix(strings.TrimSpace(bodyStr), "<!DOCTYPE") {
        return nil, fmt.Errorf("未登录或session已过期")
    }

    // 解析JSON响应
    var schedules []CourseSchedule
    if err := json.Unmarshal(body, &schedules); err != nil {
        // 可能是空对象或其他格式
        return nil, fmt.Errorf("解析响应失败: %v", err)
    }

    return schedules, nil
}

// FetchAllCourseSchedules 获取学生所有已选课程的上课时间
func (g *Grabber) FetchAllCourseSchedules() ([]CourseSchedule, error) {
    req, err := http.NewRequest("GET", GDUTBaseURL+"/xsxklist!getXsAllSksj.action?xnxqdm=202601", nil)
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %v", err)
    }

    req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    req.Header.Set("Referer", GDUTBaseURL+"/xsxklist!xsmhxsxk.action")

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("请求失败: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %v", err)
    }

    bodyStr := string(body)
    if strings.HasPrefix(strings.TrimSpace(bodyStr), "<!DOCTYPE") {
        return nil, fmt.Errorf("未登录或session已过期")
    }

    // 解析JSON响应
    var schedules []CourseSchedule
    if err := json.Unmarshal(body, &schedules); err != nil {
        return nil, fmt.Errorf("解析响应失败: %v", err)
    }

    return schedules, nil
}

// FetchSelectedCourses 获取已选课程
func (g *Grabber) FetchSelectedCourses() ([]Course, error) {
	req, err := http.NewRequest("POST", GDUTBaseURL+SelectedCourseAPI, strings.NewReader("sort=kcrwdm&order=asc&page=1&rows=500"))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", GDUTBaseURL+"/xsxklist!xsmhxkList.action")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	bodyStr := string(body)
	if strings.HasPrefix(strings.TrimSpace(bodyStr), "<!DOCTYPE") {
		return nil, fmt.Errorf("未登录或session已过期")
	}

	var withRows struct {
		Rows []json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(body, &withRows); err == nil && len(withRows.Rows) > 0 {
		return parseCoursesFromRaw(withRows.Rows)
	}

	var directArray []json.RawMessage
	if err := json.Unmarshal(body, &directArray); err == nil {
		return parseCoursesFromRaw(directArray)
	}

	return nil, fmt.Errorf("无法解析课程数据")
}

func parseCoursesFromRaw(rows []json.RawMessage) ([]Course, error) {
	var courses []Course
	for _, row := range rows {
		var raw map[string]interface{}
		if err := json.Unmarshal(row, &raw); err != nil {
			continue
		}

		course := Course{
			ID:       getString(raw, "kcrwdm"),
			Code:     getString(raw, "kcdm"),
			Name:     getString(raw, "kcmc"),
			Teacher:  getString(raw, "teaxm"),
			Category: getString(raw, "kcdlmc"),
			Type:     getString(raw, "kcflmc"),
			Credits:  getString(raw, "xf"),
			Hours:    getString(raw, "zxs"),
			Capacity: getInt(raw, "pkrs"),
			Enrolled: getInt(raw, "jxbrs"),
			Jxbdm:    getString(raw, "jxbdm"),
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

// DropCourse 退选
func (g *Grabber) DropCourse(courseID, courseName, jxbdm string) GrabResult {
	data := fmt.Sprintf("jxbdm=%s&kcrwdm=%s&kcmc=%s", jxbdm, courseID, courseName)

	req, err := http.NewRequest("POST", GDUTBaseURL+DropCourseAPI, strings.NewReader(data))
	if err != nil {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    false,
			Message:    fmt.Sprintf("创建请求失败: %v", err),
			Time:       time.Now(),
		}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "text/plain, */*; q=0.01")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", GDUTBaseURL+"/xsxklist!xsmhxsxk.action")

	resp, err := g.client.Do(req)
	if err != nil {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    false,
			Message:    fmt.Sprintf("请求失败: %v", err),
			Time:       time.Now(),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    false,
			Message:    fmt.Sprintf("读取响应失败: %v", err),
			Time:       time.Now(),
		}
	}

	ret := strings.TrimSpace(string(body))

	if ret == "1" {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    true,
			Message:    "退选成功",
			Time:       time.Now(),
		}
	}

	return GrabResult{
		CourseID:   courseID,
		CourseName: courseName,
		Success:    false,
		Message:    ret,
		Time:       time.Now(),
	}
}

// SelectCourse 选课
func (g *Grabber) SelectCourse(courseID, courseName string) GrabResult {
	data := fmt.Sprintf("kcrwdm=%s&kcmc=%s", courseID, courseName)

	req, err := http.NewRequest("POST", GDUTBaseURL+SelectCourseAPI, strings.NewReader(data))
	if err != nil {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    false,
			Message:    fmt.Sprintf("创建请求失败: %v", err),
			Time:       time.Now(),
		}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", GDUTBaseURL+"/xskjcjxx!kjcjList.action")

	resp, err := g.client.Do(req)
	if err != nil {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    false,
			Message:    fmt.Sprintf("请求失败: %v", err),
			Time:       time.Now(),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    false,
			Message:    fmt.Sprintf("读取响应失败: %v", err),
			Time:       time.Now(),
		}
	}

	ret := strings.TrimSpace(string(body))

	if ret == "1" {
		return GrabResult{
			CourseID:   courseID,
			CourseName: courseName,
			Success:    true,
			Message:    "选课成功",
			Time:       time.Now(),
		}
	}

	message := parseErrorMessage(ret)
	return GrabResult{
		CourseID:   courseID,
		CourseName: courseName,
		Success:    false,
		Message:    message,
		Time:       time.Now(),
	}
}

func parseErrorMessage(ret string) string {
	switch {
	case strings.Contains(ret, "当前不是选课时间"):
		return "当前不是选课时间"
	case strings.Contains(ret, "选课人数超出"):
		return "课程已满"
	case strings.Contains(ret, "上课时间有冲突"):
		return "上课时间有冲突"
	case strings.Contains(ret, "您已经选了该门课程"):
		return "已选该课程"
	case strings.Contains(ret, "超出选课要求门数"):
		return "超出选课要求门数"
	default:
		return ret
	}
}

// StartGrabbing 开始抢课（并发版本）
func (g *Grabber) StartGrabbing(courseIDs []string, courseNames map[string]string, interval time.Duration) {
	g.mutex.Lock()
	if g.running {
		g.mutex.Unlock()
		return
	}
	g.running = true
	g.mutex.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 使用channel控制并发
		concurrency := len(courseIDs) // 每门课一个goroutine
		if concurrency > 5 {
			concurrency = 5 // 最大并发数限制为5
		}
		sem := make(chan struct{}, concurrency)

		for {
			select {
			case <-g.stopChan:
				g.mutex.Lock()
				g.running = false
				g.mutex.Unlock()
				return
			case <-ticker.C:
				var wg sync.WaitGroup
				var mu sync.Mutex

				for _, id := range courseIDs {
					wg.Add(1)
					sem <- struct{}{} // 获取信号量

					go func(courseID string) {
						defer wg.Done()
						defer func() { <-sem }() // 释放信号量

						name := courseNames[courseID]
						result := g.SelectCourse(courseID, name)

						mu.Lock()
						g.results = append(g.results, result)
						mu.Unlock()

						if result.Success {
							fmt.Printf("[%s] ✓ 选课成功: %s\n", time.Now().Format("15:04:05"), name)
						} else {
							fmt.Printf("[%s] ✗ 选课失败: %s - %s\n", time.Now().Format("15:04:05"), name, result.Message)
						}
					}(id)
				}

				wg.Wait()

				// 重新计算剩余课程
				mu.Lock()
				newRemaining := []string{}
				for _, id := range courseIDs {
					name := courseNames[id]
					shouldKeep := true

					// 检查最新结果
					for i := len(g.results) - 1; i >= 0; i-- {
						r := g.results[i]
						if r.CourseName == name {
							if r.Success {
								shouldKeep = false
							} else {
								switch r.Message {
								case "已选该课程", "上课时间有冲突", "超出选课要求门数":
									shouldKeep = false
								}
							}
							break
						}
					}

					if shouldKeep {
						newRemaining = append(newRemaining, id)
					}
				}
				courseIDs = newRemaining
				mu.Unlock()

				if len(courseIDs) == 0 {
					fmt.Printf("[%s] 所有课程已选完\n", time.Now().Format("15:04:05"))
					g.mutex.Lock()
					g.running = false
					g.mutex.Unlock()
					return
				}
			}
		}
	}()
}

func (g *Grabber) StopGrabbing() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if g.running {
		g.stopChan <- true
	}
}

func (g *Grabber) GetResults() []GrabResult {
	return g.results
}

func (g *Grabber) ClearResults() {
	g.results = nil
}

func (g *Grabber) IsRunning() bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.running
}
