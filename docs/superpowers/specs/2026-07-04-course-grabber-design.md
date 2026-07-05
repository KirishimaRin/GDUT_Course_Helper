# 抢课系统设计文档

## 1. 项目概述

### 1.1 项目目标
开发一个基于学校教务系统的抢课系统，实现自动抢课、多课程并发、自动确认等功能。

### 1.2 核心需求
- 自动登录教务系统（账号密码）
- 从教务系统获取课程列表
- 用户选择要抢的课程
- 在选课时间开放时立即抢课
- 支持多课程并发抢课
- 网络错误自动重试
- 课程已满时按心愿清单顺序继续抢课
- Web界面实时显示状态
- 单用户使用
- 本地运行

### 1.3 技术选型
- **后端**：Go + Gin框架
- **前端**：原生HTML/CSS/JavaScript
- **数据库**：SQLite
- **通信**：HTTP + WebSocket

## 2. 系统架构

### 2.1 整体架构
```
┌─────────────────────────────────────────────────┐
│                   Web浏览器                      │
│  (HTML/CSS/JavaScript)                          │
└─────────────────────────────────────────────────┘
                    │ HTTP/WebSocket
                    ▼
┌─────────────────────────────────────────────────┐
│              Go Web服务器 (Gin)                  │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │  认证模块   │ │  课程模块   │ │  抢课模块   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  │  调度模块   │ │  通知模块   │ │  存储模块   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ │
└─────────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│              SQLite数据库                        │
│  (用户配置、课程列表、抢课记录)                  │
└─────────────────────────────────────────────────┘
```

### 2.2 模块职责
1. **认证模块**：处理教务系统登录，管理Cookie/Session
2. **课程模块**：从教务系统获取课程列表，解析课程信息
3. **抢课模块**：执行抢课逻辑，支持并发抢课
4. **调度模块**：管理抢课任务调度，定时抢课和监控
5. **通知模块**：通过WebSocket实时推送状态更新
6. **存储模块**：管理SQLite数据库，存储配置和记录

## 3. 核心模块设计

### 3.1 认证模块

**功能**：处理教务系统登录，管理用户会话

**登录流程**：
1. 用户输入教务系统账号密码
2. 模拟浏览器登录请求，获取Cookie/Session
3. 验证登录状态，保存会话信息
4. 定期检查会话有效性，自动刷新

**数据结构**：
```go
type UserCredentials struct {
    Username string `json:"username"`
    Password string `json:"password"`
    LoginURL string `json:"login_url"`
}

type Session struct {
    Cookies  []*http.Cookie `json:"cookies"`
    Token    string         `json:"token"`
    Expires  time.Time      `json:"expires"`
}
```

### 3.2 课程模块

**功能**：获取课程列表，解析课程信息

**课程获取流程**：
1. 使用有效会话请求课程列表API
2. 解析HTML/JSON响应，提取课程信息
3. 过滤可选课程，显示给用户选择
4. 支持按课程号、课程名、教师等筛选

**数据结构**：
```go
type Course struct {
    ID        string `json:"id"`        // 课程ID
    Code      string `json:"code"`      // 课程代码
    Name      string `json:"name"`      // 课程名称
    Teacher   string `json:"teacher"`   // 教师
    Time      string `json:"time"`      // 上课时间
    Location  string `json:"location"`  // 上课地点
    Capacity  int    `json:"capacity"`  // 课程容量
    Enrolled  int    `json:"enrolled"`  // 已选人数
    Available int    `json:"available"` // 剩余名额
    Status    string `json:"status"`    // 课程状态
}
```

### 3.3 抢课模块

**功能**：执行抢课逻辑，支持并发抢课

**抢课流程**：
1. 接收用户选择的课程列表（心愿清单）
2. 按优先级排序课程
3. 启动多个goroutine并发抢课
4. 网络错误自动重试（指数退避）
5. 课程已满时按心愿清单顺序继续抢课
6. 直到抢到用户所需的课程数量

**并发控制**：
```go
type CourseGrabber struct {
    Concurrency int              // 并发数
    Courses     []Course         // 课程列表
    Results     chan GrabResult   // 结果通道
    Done        chan bool         // 完成信号
}

type GrabResult struct {
    Course  Course
    Success bool
    Error   error
    Time    time.Time
}
```

### 3.4 调度模块

**功能**：管理抢课任务调度

**调度策略**：
1. **定时抢课**：在指定时间自动开始抢课
2. **监控抢课**：持续监控课程名额，有空位立即抢
3. **混合策略**：先定时，时间到后自动切换为监控模式

**混合策略设计意图**：
- 确保在选课开放的第一时间抢课
- 处理课程已满的情况，继续监控直到抢到课程
- 避免在选课时间前频繁请求教务系统，减少被封风险

**任务管理**：
```go
type GrabTask struct {
    ID          string        `json:"id"`
    Courses     []Course      `json:"courses"`
    Strategy    string        `json:"strategy"`    // "scheduled", "monitor", "hybrid"
    ScheduledAt time.Time     `json:"scheduled_at"` // 定时时间
    Status      string        `json:"status"`       // "pending", "running", "completed", "failed"
    CreatedAt   time.Time     `json:"created_at"`
}
```

### 3.5 通知模块

**功能**：通过WebSocket实时推送状态更新

**WebSocket协议**：
```json
{
    "type": "status_update",
    "data": {
        "task_id": "task_123",
        "course_id": "course_456",
        "status": "grabbing", // grabbing, success, failed, waiting
        "message": "正在抢课：高等数学A",
        "timestamp": "2024-01-01T12:00:00Z"
    }
}
```

**消息类型**：
- `task_created`：任务创建成功
- `task_started`：任务开始执行
- `course_grabbing`：正在抢课
- `course_success`：抢课成功
- `course_failed`：抢课失败
- `task_completed`：任务完成

### 3.6 存储模块

**功能**：管理SQLite数据库

**数据库表设计**：
```sql
-- 用户配置表
CREATE TABLE user_config (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    login_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 课程表
CREATE TABLE courses (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    teacher TEXT,
    time TEXT,
    location TEXT,
    capacity INTEGER,
    enrolled INTEGER,
    status TEXT,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 抢课任务表
CREATE TABLE grab_tasks (
    id TEXT PRIMARY KEY,
    strategy TEXT NOT NULL,
    scheduled_at DATETIME,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

-- 任务课程关联表
CREATE TABLE task_courses (
    task_id TEXT,
    course_id TEXT,
    priority INTEGER,
    status TEXT DEFAULT 'pending',
    grabbed_at DATETIME,
    PRIMARY KEY (task_id, course_id),
    FOREIGN KEY (task_id) REFERENCES grab_tasks(id),
    FOREIGN KEY (course_id) REFERENCES courses(id)
);
```

## 4. 数据流设计

### 4.1 主要数据流
```
用户配置 → 认证模块 → 课程模块 → 抢课模块 → 通知模块 → Web界面
    ↓          ↓          ↓          ↓          ↓
SQLite ← 会话管理 ← 课程列表 ← 抢课结果 ← 状态更新 ← 实时显示
```

### 4.2 详细数据流

**用户配置流程**：
1. 用户通过Web界面输入教务系统账号密码
2. 认证模块验证登录，保存会话到SQLite
3. 课程模块获取课程列表，显示给用户选择

**抢课任务流程**：
1. 用户选择要抢的课程，创建抢课任务
2. 调度模块根据策略安排任务执行
3. 抢课模块执行抢课，结果写入SQLite
4. 通知模块通过WebSocket推送状态到Web界面

**状态同步流程**：
1. 抢课模块定期检查任务状态
2. 更新SQLite中的任务和课程状态
3. 通知模块实时推送状态变化

## 5. 错误处理设计

### 5.1 错误类型和处理策略

**网络错误**：
- **处理**：自动重试，指数退避（1s, 2s, 4s, 8s...）
- **最大重试**：5次
- **超时**：单个请求30秒超时
- **通知**：Web界面显示重试状态

**认证错误**：
- **处理**：提示用户重新登录
- **会话过期**：自动尝试刷新会话
- **通知**：Web界面显示认证状态

**课程已满**：
- **处理**：按心愿清单顺序继续抢课
- **监控**：继续监控已满课程，等待名额释放
- **通知**：Web界面显示课程状态

**系统错误**：
- **处理**：记录错误日志，继续执行其他任务
- **恢复**：系统重启后从SQLite恢复任务状态
- **通知**：Web界面显示错误信息

### 5.2 错误处理代码示例
```go
type ErrorHandler struct {
    MaxRetries int
    BaseDelay  time.Duration
}

func (e *ErrorHandler) HandleError(err error, task *GrabTask) {
    switch {
    case isNetworkError(err):
        e.handleNetworkError(err, task)
    case isAuthError(err):
        e.handleAuthError(err, task)
    case isCourseFullError(err):
        e.handleCourseFullError(err, task)
    default:
        e.handleSystemError(err, task)
    }
}

func (e *ErrorHandler) handleNetworkError(err error, task *GrabTask) {
    for i := 0; i < e.MaxRetries; i++ {
        delay := e.BaseDelay * time.Duration(1<<uint(i))
        time.Sleep(delay)
        if retrySuccess(task) {
            return
        }
    }
    // 重试失败，记录错误
    log.Printf("Network error after %d retries: %v", e.MaxRetries, err)
    notifyWebUI(task, "网络错误，重试失败")
}
```

## 6. 接口设计

### 6.1 REST API

**认证相关**：
- `POST /api/login` - 用户登录
- `GET /api/session` - 获取会话状态
- `POST /api/logout` - 用户登出

**课程相关**：
- `GET /api/courses` - 获取课程列表
- `GET /api/courses/:id` - 获取课程详情

**任务相关**：
- `POST /api/tasks` - 创建抢课任务
- `GET /api/tasks` - 获取任务列表
- `GET /api/tasks/:id` - 获取任务详情
- `DELETE /api/tasks/:id` - 删除任务

**WebSocket**：
- `GET /ws` - WebSocket连接，实时接收状态更新

### 6.2 前端页面

**主要页面**：
1. **登录页面**：输入教务系统账号密码
2. **课程列表页面**：显示可选课程，支持筛选
3. **抢课任务页面**：创建和管理抢课任务
4. **任务状态页面**：实时显示抢课状态和结果

### 6.3 前端设计系统

**设计模式**：Minimal Single Column
- 单列布局，大字体，大量留白
- 移动优先设计
- 单一CTA焦点，减少导航干扰

**视觉风格**：Vibrant & Block-based
- 大胆、充满活力的设计风格
- 块状布局，几何形状
- 高色彩对比度，双色调效果

**配色方案**：
```css
:root {
  --color-primary: #171717;      /* 主色 - 深黑 */
  --color-on-primary: #FFFFFF;   /* 主色上的文字 - 白色 */
  --color-secondary: #404040;    /* 次要色 - 深灰 */
  --color-accent: #A16207;       /* 强调色 - 金色 */
  --color-background: #FFFFFF;   /* 背景色 - 白色 */
  --color-foreground: #171717;   /* 前景色 - 深黑 */
  --color-muted: #E8ECF0;        /* 柔和色 - 浅灰 */
  --color-border: #E5E5E5;       /* 边框色 - 浅灰 */
  --color-destructive: #DC2626;  /* 危险色 - 红色 */
  --color-ring: #171717;         /* 焦点环色 - 深黑 */
}
```

**字体方案**：
```css
/* 主字体：Plus Jakarta Sans */
@import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:ital,wght@0,400;0,500;0,700;1,400&display=swap');

:root {
  --font-sans: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
}
```

**间距系统**：
```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;
}
```

**组件规范**：
- **按钮**：大尺寸（48px高度），圆角8px，悬停时颜色变化
- **输入框**：高度48px，圆角8px，聚焦时有焦点环
- **卡片**：圆角12px，阴影效果，悬停时轻微上移
- **表格**：紧凑行高，交替行颜色，悬停高亮

**响应式断点**：
```css
/* 移动端 */
@media (max-width: 375px) { ... }

/* 平板端 */
@media (min-width: 768px) { ... }

/* 桌面端 */
@media (min-width: 1024px) { ... }

/* 大屏桌面 */
@media (min-width: 1440px) { ... }
```

**动画效果**：
- **悬停效果**：150-300ms过渡，颜色变化或轻微上移
- **加载动画**：旋转或脉冲动画
- **状态转换**：平滑的淡入淡出效果
- **滚动动画**：元素进入视口时的淡入效果

**无障碍设计**：
- **对比度**：文字对比度≥4.5:1
- **焦点状态**：键盘导航时清晰可见
- **屏幕阅读器**：所有交互元素有适当的标签
- **减少动画**：尊重`prefers-reduced-motion`设置

## 7. 部署和运行

### 7.1 环境要求
- Go 1.21+
- SQLite3
- 现代浏览器（Chrome, Firefox, Safari, Edge）

### 7.2 运行方式
```bash
# 编译
go build -o course-grabber

# 运行
./course-grabber

# 访问Web界面
# http://localhost:8080
```

### 7.3 配置文件
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

## 8. 安全考虑

### 8.1 数据安全
- 用户密码加密存储（bcrypt）
- 会话信息本地存储，不上传到外部服务器
- SQLite数据库文件权限控制

### 8.2 网络安全
- HTTPS支持（可选）
- 请求频率限制，避免被封
- User-Agent随机化，模拟正常浏览器

### 8.3 系统安全
- 输入验证和过滤
- SQL注入防护
- XSS防护

## 9. 测试策略

### 9.1 单元测试
- 认证模块测试
- 课程模块测试
- 抢课模块测试
- 调度模块测试

### 9.2 集成测试
- API接口测试
- WebSocket通信测试
- 数据库操作测试

### 9.3 端到端测试
- 完整抢课流程测试
- 错误处理测试
- 并发性能测试

## 10. 后续扩展

### 10.1 功能扩展
- 支持更多教务系统
- 添加课程监控提醒
- 支持课程冲突检测
- 添加抢课统计和分析

### 10.2 性能优化
- 连接池优化
- 请求缓存
- 数据库索引优化
- 并发性能调优

### 10.3 用户体验
- 移动端适配
- 主题切换
- 多语言支持
- 操作引导和帮助

## 11. 总结

本设计文档详细描述了抢课系统的架构、模块设计、数据流、错误处理、接口设计、部署运行、安全考虑、测试策略和后续扩展。系统采用Go + Gin + SQLite的技术栈，支持多课程并发抢课，通过WebSocket实时推送状态，满足用户在选课时间开放时立即抢课的需求。

设计重点：
1. **简单高效**：单体架构，本地运行，易于开发和部署
2. **并发抢课**：支持多课程并发，提高抢课成功率
3. **智能调度**：混合策略，先定时后监控，确保第一时间抢课
4. **实时通知**：WebSocket实时推送，Web界面实时显示状态
5. **错误处理**：网络错误自动重试，课程已满继续监控
6. **数据安全**：本地存储，加密保护，安全可靠