# 抢课系统实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 开发一个基于学校教务系统的抢课系统，实现自动抢课、多课程并发、自动确认等功能。

**Architecture:** 采用Go + Gin框架的单体架构，SQLite数据库，原生HTML/CSS/JavaScript前端，WebSocket实时通信。

**Tech Stack:** Go, Gin, SQLite, HTML/CSS/JavaScript, WebSocket

## Global Constraints

- 使用Go 1.21+版本
- 使用SQLite3作为数据库
- 支持现代浏览器（Chrome, Firefox, Safari, Edge）
- 本地运行，单用户使用
- 前端使用原生HTML/CSS/JavaScript，不使用框架
- 使用Plus Jakarta Sans字体
- 配色方案：主色#171717，强调色#A16207

---

## 文件结构

```
E:\Project\Course_helper\
├── cmd/
│   └── server/
│       └── main.go                 # 主程序入口
├── internal/
│   ├── auth/
│   │   ├── auth.go                 # 认证模块
│   │   └── auth_test.go            # 认证模块测试
│   ├── course/
│   │   ├── course.go               # 课程模块
│   │   └── course_test.go          # 课程模块测试
│   ├── grabber/
│   │   ├── grabber.go              # 抢课模块
│   │   └── grabber_test.go         # 抢课模块测试
│   ├── scheduler/
│   │   ├── scheduler.go            # 调度模块
│   │   └── scheduler_test.go       # 调度模块测试
│   ├── notifier/
│   │   ├── notifier.go             # 通知模块
│   │   └── notifier_test.go        # 通知模块测试
│   ├── storage/
│   │   ├── storage.go              # 存储模块
│   │   └── storage_test.go         # 存储模块测试
│   └── handler/
│       ├── handler.go              # HTTP处理器
│       └── handler_test.go         # HTTP处理器测试
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css           # 样式文件
│   │   └── js/
│   │       └── app.js              # 前端JavaScript
│   └── templates/
│       ├── index.html              # 主页面模板
│       ├── login.html              # 登录页面模板
│       ├── courses.html            # 课程列表页面模板
│       ├── tasks.html              # 任务管理页面模板
│       └── status.html             # 状态页面模板
├── data/
│   └── course-grabber.db           # SQLite数据库文件
├── config.yaml                     # 配置文件
├── go.mod                          # Go模块文件
└── go.sum                          # Go依赖文件
```

---

## 实施任务

### Task 1: 项目初始化和基础设置

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `config.yaml`

**Interfaces:**
- Produces: 项目基础结构，Go模块初始化

- [ ] **Step 1: 初始化Go模块**

```bash
cd E:\Project\Course_helper
go mod init course-grabber
```

- [ ] **Step 2: 创建配置文件**

```yaml
# config.yaml
server:
  port: 8080
  host: localhost

database:
  path: ./data/course-grabber.db

grabber:
  concurrency: 3
  max_retries: 5
  base_delay: 1s
  request_timeout: 30s
```

- [ ] **Step 3: 创建主程序入口**

```go
// cmd/server/main.go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
)

func main() {
    // 创建必要的目录
    os.MkdirAll("data", 0755)
    os.MkdirAll("web/static/css", 0755)
    os.MkdirAll("web/static/js", 0755)
    os.MkdirAll("web/templates", 0755)

    // 初始化Gin路由
    r := gin.Default()

    // 加载HTML模板
    r.LoadHTMLGlob("web/templates/*")

    // 静态文件服务
    r.Static("/static", "./web/static")

    // 首页路由
    r.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    // 启动服务器
    log.Println("Server starting on :8080")
    r.Run(":8080")
}
```

- [ ] **Step 4: 安装依赖**

```bash
go get github.com/gin-gonic/gin
go get github.com/mattn/go-sqlite3
```

- [ ] **Step 5: 验证项目结构**

```bash
go build ./cmd/server/
```

- [ ] **Step 6: 提交代码**

```bash
git init
git add .
git commit -m "feat: initialize project structure"
```

---

### Task 2: 存储模块实现

**Files:**
- Create: `internal/storage/storage.go`
- Create: `internal/storage/storage_test.go`

**Interfaces:**
- Produces: `Storage` struct with methods: `Init()`, `Close()`, `SaveUserConfig()`, `GetUserConfig()`, `SaveCourse()`, `GetCourses()`, `SaveTask()`, `GetTasks()`, `UpdateTaskStatus()`

- [ ] **Step 1: 编写存储模块测试**

```go
// internal/storage/storage_test.go
package storage

import (
    "os"
    "testing"
)

func TestStorageInit(t *testing.T) {
    dbPath := "./test.db"
    defer os.Remove(dbPath)

    s := New(dbPath)
    err := s.Init()
    if err != nil {
        t.Fatalf("Failed to initialize storage: %v", err)
    }
    defer s.Close()

    var count int
    err = s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table'").Scan(&count)
    if err != nil {
        t.Fatalf("Failed to query tables: %v", err)
    }

    if count < 4 {
        t.Errorf("Expected at least 4 tables, got %d", count)
    }
}

func TestSaveAndGetUserConfig(t *testing.T) {
    dbPath := "./test_config.db"
    defer os.Remove(dbPath)

    s := New(dbPath)
    s.Init()
    defer s.Close()

    config := &UserConfig{
        Username: "testuser",
        LoginURL: "http://example.com/login",
    }

    err := s.SaveUserConfig(config)
    if err != nil {
        t.Fatalf("Failed to save user config: %v", err)
    }

    saved, err := s.GetUserConfig()
    if err != nil {
        t.Fatalf("Failed to get user config: %v", err)
    }

    if saved.Username != config.Username {
        t.Errorf("Expected username %s, got %s", config.Username, saved.Username)
    }
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/storage/ -v
```

- [ ] **Step 3: 实现存储模块**

```go
// internal/storage/storage.go
package storage

import (
    "database/sql"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type UserConfig struct {
    ID        int       `json:"id"`
    Username  string    `json:"username"`
    LoginURL  string    `json:"login_url"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Course struct {
    ID        string    `json:"id"`
    Code      string    `json:"code"`
    Name      string    `json:"name"`
    Teacher   string    `json:"teacher"`
    Time      string    `json:"time"`
    Location  string    `json:"location"`
    Capacity  int       `json:"capacity"`
    Enrolled  int       `json:"enrolled"`
    Available int       `json:"available"`
    Status    string    `json:"status"`
    FetchedAt time.Time `json:"fetched_at"`
}

type Task struct {
    ID          string     `json:"id"`
    Strategy    string     `json:"strategy"`
    ScheduledAt *time.Time `json:"scheduled_at"`
    Status      string     `json:"status"`
    CreatedAt   time.Time  `json:"created_at"`
    CompletedAt *time.Time `json:"completed_at"`
}

type TaskCourse struct {
    TaskID    string     `json:"task_id"`
    CourseID  string     `json:"course_id"`
    Priority  int        `json:"priority"`
    Status    string     `json:"status"`
    GrabbedAt *time.Time `json:"grabbed_at"`
}

type Storage struct {
    dbPath string
    db     *sql.DB
}

func New(dbPath string) *Storage {
    return &Storage{dbPath: dbPath}
}

func (s *Storage) Init() error {
    db, err := sql.Open("sqlite3", s.dbPath)
    if err != nil {
        return err
    }
    s.db = db

    queries := []string{
        `CREATE TABLE IF NOT EXISTS user_config (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL,
            login_url TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
        `CREATE TABLE IF NOT EXISTS courses (
            id TEXT PRIMARY KEY,
            code TEXT NOT NULL,
            name TEXT NOT NULL,
            teacher TEXT,
            time TEXT,
            location TEXT,
            capacity INTEGER,
            enrolled INTEGER,
            available INTEGER,
            status TEXT,
            fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
        `CREATE TABLE IF NOT EXISTS grab_tasks (
            id TEXT PRIMARY KEY,
            strategy TEXT NOT NULL,
            scheduled_at DATETIME,
            status TEXT DEFAULT 'pending',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            completed_at DATETIME
        )`,
        `CREATE TABLE IF NOT EXISTS task_courses (
            task_id TEXT,
            course_id TEXT,
            priority INTEGER,
            status TEXT DEFAULT 'pending',
            grabbed_at DATETIME,
            PRIMARY KEY (task_id, course_id),
            FOREIGN KEY (task_id) REFERENCES grab_tasks(id),
            FOREIGN KEY (course_id) REFERENCES courses(id)
        )`,
    }

    for _, query := range queries {
        _, err = s.db.Exec(query)
        if err != nil {
            return err
        }
    }

    return nil
}

func (s *Storage) Close() error {
    if s.db != nil {
        return s.db.Close()
    }
    return nil
}

func (s *Storage) SaveUserConfig(config *UserConfig) error {
    query := `INSERT OR REPLACE INTO user_config (username, login_url, updated_at) 
              VALUES (?, ?, CURRENT_TIMESTAMP)`
    _, err := s.db.Exec(query, config.Username, config.LoginURL)
    return err
}

func (s *Storage) GetUserConfig() (*UserConfig, error) {
    config := &UserConfig{}
    query := `SELECT id, username, login_url, created_at, updated_at 
              FROM user_config ORDER BY id DESC LIMIT 1`
    err := s.db.QueryRow(query).Scan(
        &config.ID, &config.Username, &config.LoginURL,
        &config.CreatedAt, &config.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return config, nil
}

func (s *Storage) SaveCourse(course *Course) error {
    query := `INSERT OR REPLACE INTO courses 
              (id, code, name, teacher, time, location, capacity, enrolled, available, status, fetched_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
    _, err := s.db.Exec(query,
        course.ID, course.Code, course.Name, course.Teacher,
        course.Time, course.Location, course.Capacity,
        course.Enrolled, course.Available, course.Status,
    )
    return err
}

func (s *Storage) GetCourses() ([]*Course, error) {
    query := `SELECT id, code, name, teacher, time, location, capacity, enrolled, available, status, fetched_at 
              FROM courses ORDER BY fetched_at DESC`
    rows, err := s.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var courses []*Course
    for rows.Next() {
        c := &Course{}
        err := rows.Scan(
            &c.ID, &c.Code, &c.Name, &c.Teacher,
            &c.Time, &c.Location, &c.Capacity,
            &c.Enrolled, &c.Available, &c.Status, &c.FetchedAt,
        )
        if err != nil {
            return nil, err
        }
        courses = append(courses, c)
    }
    return courses, nil
}

func (s *Storage) SaveTask(task *Task) error {
    query := `INSERT OR REPLACE INTO grab_tasks (id, strategy, scheduled_at, status, created_at)
              VALUES (?, ?, ?, ?, ?)`
    _, err := s.db.Exec(query,
        task.ID, task.Strategy, task.ScheduledAt,
        task.Status, task.CreatedAt,
    )
    return err
}

func (s *Storage) GetTasks() ([]*Task, error) {
    query := `SELECT id, strategy, scheduled_at, status, created_at, completed_at 
              FROM grab_tasks ORDER BY created_at DESC`
    rows, err := s.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tasks []*Task
    for rows.Next() {
        t := &Task{}
        err := rows.Scan(
            &t.ID, &t.Strategy, &t.ScheduledAt,
            &t.Status, &t.CreatedAt, &t.CompletedAt,
        )
        if err != nil {
            return nil, err
        }
        tasks = append(tasks, t)
    }
    return tasks, nil
}

func (s *Storage) UpdateTaskStatus(taskID, status string) error {
    query := `UPDATE grab_tasks SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`
    _, err := s.db.Exec(query, status, taskID)
    return err
}

func (s *Storage) SaveTaskCourse(tc *TaskCourse) error {
    query := `INSERT OR REPLACE INTO task_courses (task_id, course_id, priority, status)
              VALUES (?, ?, ?, ?)`
    _, err := s.db.Exec(query, tc.TaskID, tc.CourseID, tc.Priority, tc.Status)
    return err
}

func (s *Storage) GetTaskCourses(taskID string) ([]*TaskCourse, error) {
    query := `SELECT task_id, course_id, priority, status, grabbed_at 
              FROM task_courses WHERE task_id = ? ORDER BY priority`
    rows, err := s.db.Query(query, taskID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var taskCourses []*TaskCourse
    for rows.Next() {
        tc := &TaskCourse{}
        err := rows.Scan(
            &tc.TaskID, &tc.CourseID, &tc.Priority,
            &tc.Status, &tc.GrabbedAt,
        )
        if err != nil {
            return nil, err
        }
        taskCourses = append(taskCourses, tc)
    }
    return taskCourses, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/storage/ -v
```

- [ ] **Step 5: 提交代码**

```bash
git add internal/storage/
git commit -m "feat: implement storage module"
```

---

### Task 3: 认证模块实现

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

**Interfaces:**
- Consumes: `Storage` from Task 2
- Produces: `Auth` struct with methods: `Login()`, `Logout()`, `GetSession()`, `IsLoggedIn()`

- [ ] **Step 1: 编写认证模块测试**

```go
// internal/auth/auth_test.go
package auth

import (
    "testing"
    "course-grabber/internal/storage"
)

func TestAuthLogin(t *testing.T) {
    s := storage.New("./test_auth.db")
    s.Init()
    defer s.Close()

    a := New(s)
    
    config := &storage.UserConfig{
        Username: "testuser",
        LoginURL: "http://example.com/login",
    }
    
    err := s.SaveUserConfig(config)
    if err != nil {
        t.Fatalf("Failed to save config: %v", err)
    }
    
    if a.IsLoggedIn() {
        t.Error("Expected not logged in initially")
    }
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/auth/ -v
```

- [ ] **Step 3: 实现认证模块**

```go
// internal/auth/auth.go
package auth

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "net/http"
    "time"

    "course-grabber/internal/storage"
)

type Session struct {
    Cookies []*http.Cookie `json:"cookies"`
    Token   string         `json:"token"`
    Expires time.Time      `json:"expires"`
}

type Auth struct {
    storage *storage.Storage
    session *Session
}

func New(storage *storage.Storage) *Auth {
    return &Auth{
        storage: storage,
    }
}

func (a *Auth) Login(username, password, loginURL string) error {
    token, err := generateToken()
    if err != nil {
        return err
    }

    a.session = &Session{
        Token:   token,
        Expires: time.Now().Add(24 * time.Hour),
    }

    config := &storage.UserConfig{
        Username: username,
        LoginURL: loginURL,
    }
    err = a.storage.SaveUserConfig(config)
    if err != nil {
        return err
    }

    return nil
}

func (a *Auth) Logout() {
    a.session = nil
}

func (a *Auth) GetSession() *Session {
    if a.session == nil {
        return nil
    }
    
    if time.Now().After(a.session.Expires) {
        a.session = nil
        return nil
    }
    
    return a.session
}

func (a *Auth) IsLoggedIn() bool {
    return a.GetSession() != nil
}

func (a *Auth) RefreshSession() error {
    if a.session == nil {
        return errors.New("no active session")
    }
    
    a.session.Expires = time.Now().Add(24 * time.Hour)
    return nil
}

func generateToken() (string, error) {
    bytes := make([]byte, 32)
    _, err := rand.Read(bytes)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/auth/ -v
```

- [ ] **Step 5: 提交代码**

```bash
git add internal/auth/
git commit -m "feat: implement auth module"
```

---

## 继续实施

由于实施计划较长，这里只展示了前3个任务的详细步骤。完整的实施计划包括以下任务：

4. **课程模块实现** - 获取课程列表，解析课程信息
5. **抢课模块实现** - 执行抢课逻辑，支持并发抢课
6. **调度模块实现** - 管理抢课任务调度
7. **通知模块实现** - 通过WebSocket实时推送状态更新
8. **HTTP处理器实现** - REST API和WebSocket路由
9. **前端页面实现** - HTML/CSS/JavaScript界面
10. **主程序集成** - 组装所有模块
11. **文档和测试** - README和使用说明

## 执行选项

**计划已保存。两种执行选项：**

**1. 子代理驱动（推荐）** - 我为每个任务分派新的子代理，任务之间进行审查，快速迭代

**2. 内联执行** - 在当前会话中执行任务，使用执行计划，批量执行并设置检查点

**选择哪种方式？**