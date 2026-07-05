# GDUT 抢课助手

广东工业大学教务系统自动抢课工具，支持课程浏览、收藏、并发抢课。

## 功能特性

- JSESSIONID 登录（从浏览器复制）
- 课程列表浏览，支持搜索和分类筛选
- 课程详情查看（上课时间、周次）
- 收藏夹管理（拖拽排序）
- 并发抢课（最多 5 个 goroutine）
- 已选课程管理（退选）
- 实时抢课日志

## 快速开始

### 方式一：直接下载（推荐）

从 [Releases](https://github.com/KirishimaRin/GDUT_Course_helper/releases) 下载对应平台的压缩包：

- `course-helper-v1.0.0-windows-amd64.zip` - Windows
- `course-helper-v1.0.0-linux-amd64.tar.gz` - Linux
- `course-helper-v1.0.0-darwin-amd64.tar.gz` - macOS

解压后运行：
```bash
# Windows
course-helper.exe

# Linux/macOS
./course-helper
```

### 方式二：源码编译

#### 环境要求

- Go 1.21+
- 现代浏览器

#### 安装运行

```bash
# 克隆项目
git clone https://github.com/KirishimaRin/GDUT_Course_helper.git
cd GDUT_Course_helper

# 复制配置文件
cp config.yaml.example config.yaml

# 安装依赖
go mod tidy

# 使用 Makefile（推荐）
make run

# 或手动编译运行
go build -o course-helper.exe ./cmd/server/
./course-helper.exe
```

### 访问

打开浏览器访问 http://localhost:32555

## 使用说明

1. 打开 GDUT 教务系统 (https://jxfw.gdut.edu.cn)，登录后按 F12 打开开发者工具
2. 在 Network 面板找到任意请求，复制 Cookie 中的 `JSESSIONID` 值
3. 在本系统导入页面粘贴 JSESSIONID
4. 浏览课程，点击分类标签筛选，输入关键词搜索
5. 将想抢的课程加入收藏夹
6. 点击「开始抢课」，系统自动并发抢课
7. 抢到的课程会出现在右侧已选列表

## 项目结构

```
Course_helper/
├── .github/workflows/      # GitHub Actions 自动构建
├── cmd/server/              # 主程序入口
├── internal/
│   ├── auth/               # JSESSIONID 认证
│   ├── grabber/            # 抢课核心逻辑
│   └── handler/            # HTTP 处理器
├── web/
│   ├── static/css/         # 样式文件
│   └── templates/          # HTML 模板
├── config.yaml.example     # 配置文件示例
├── Makefile                # 构建命令
└── README.md
```

## API 接口

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/login | 导入 JSESSIONID |
| POST | /api/logout | 退出登录 |
| GET | /api/status | 登录状态 |

### 课程
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/courses | 课程列表（分页） |
| GET | /api/courses/selected | 已选课程 |
| POST | /api/courses/drop | 退选课程 |
| GET | /api/courses/schedule | 课程上课时间 |
| GET | /api/courses/schedules | 所有已选课程时间 |

### 抢课
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/grab/start | 开始抢课 |
| POST | /api/grab/stop | 停止抢课 |
| GET | /api/grab/status | 抢课状态 |
| POST | /api/grab/clear | 清空结果 |

## 抢课逻辑

- 并发数：最多 5 个 goroutine 同时抢课
- 课程已满：自动跳过，继续尝试
- 时间冲突：自动移除该课程
- 超出限选：自动移除该课程
- 已选课程：自动移除

## 开发命令

```bash
make run          # 本地运行
make build        # 编译当前平台
make build-all    # 交叉编译所有平台
make clean        # 清理构建产物
```

## 免责声明

本工具**仅供学习参考**，请在下载后 **24 小时内删除**。

使用者需自行承担以下责任：

1. **合规性**：请遵守广东工业大学教务系统相关管理规定及国家法律法规
2. **使用风险**：因使用本工具导致的账号封禁、成绩作废等后果由使用者自行承担
3. **禁止商用**：严禁将本工具用于任何商业目的或收费服务
4. **责任限制**：作者不对因使用本工具产生的任何直接或间接损失承担责任

如您不同意上述条款，请立即停止使用并删除本工具。


