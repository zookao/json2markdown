package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"json2md/converter"

	"github.com/gin-gonic/gin"
)

type ConvertRequest struct {
	JSON string `json:"json" binding:"required"`
}

type ConvertResponse struct {
	Markdown string `json:"markdown"`
	Error    string `json:"error,omitempty"`
}

func main() {
	r := gin.Default()

	// 加载静态模板
	r.LoadHTMLGlob("templates/*")

	// 首页
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// API转换接口
	r.POST("/api/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ConvertResponse{
				Error: "请求参数错误: " + err.Error(),
			})
			return
		}

		markdown, err := converter.ConvertJSONToMarkdown(req.JSON)
		if err != nil {
			c.JSON(http.StatusBadRequest, ConvertResponse{
				Error: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, ConvertResponse{
			Markdown: markdown,
		})
	})

	// 启动服务前先打开浏览器
	go func() {
		openBrowser("http://localhost:8080")
	}()

	// 启动服务
	r.Run(":8080")
}

// openBrowser 打开默认浏览器访问指定URL
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("无法自动打开浏览器: %v\n", err)
	}
}
