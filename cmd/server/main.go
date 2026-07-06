package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"course-grabber/internal/handler"

	"github.com/gin-gonic/gin"
)

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}

var version = "dev"

func newRouter() *gin.Engine {
	gdutH := handler.NewGDUTHandler()

	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	r.GET("/import", func(c *gin.Context) {
		c.HTML(200, "import.html", nil)
	})
	r.GET("/courses", func(c *gin.Context) {
		c.HTML(200, "courses.html", nil)
	})

	api := r.Group("/api")
	{
		api.POST("/jsessionid", gdutH.SetJSESSIONID)
		api.GET("/status", gdutH.GetStatus)
		api.POST("/logout", gdutH.Logout)
		api.GET("/courses", gdutH.GetCourses)
		api.GET("/courses/selected", gdutH.GetSelectedCourses)
		api.POST("/courses/drop", gdutH.DropCourse)
		api.GET("/courses/pageinfo", gdutH.GetPageInfo)
		api.GET("/courses/schedule", gdutH.GetCourseSchedule)
		api.GET("/courses/schedules", gdutH.GetAllSchedules)
		api.POST("/grab/start", gdutH.StartGrabbing)
		api.POST("/grab/stop", gdutH.StopGrabbing)
		api.GET("/grab/status", gdutH.GetGrabStatus)
		api.POST("/grab/clear", gdutH.ClearResults)
	}

	return r
}

func main() {
	os.MkdirAll("web/static/css", 0755)
	os.MkdirAll("web/static/js", 0755)
	os.MkdirAll("web/templates", 0755)

	r := newRouter()

	port := ":32555"
	log.Printf("Course Helper %s", version)
	log.Printf("Server starting on http://localhost%s", port)

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost" + port)
	}()

	srv := &http.Server{Addr: port, Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Press Enter to stop the server")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
