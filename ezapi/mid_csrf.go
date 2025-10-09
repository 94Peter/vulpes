package ezapi

import (
	"net/http"

	"github.com/arwoosa/vulpes/log"
	"github.com/gin-gonic/gin"
)

func csrfProtectionWithExclusion(csrfProtector func(http.Handler) http.Handler, excludedPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 檢查目前路徑是否在排除列表中
		for _, path := range excludedPaths {
			if c.Request.URL.Path == path {
				// 如果在列表中，直接呼叫下一個處理程序，不進行 CSRF 驗證
				c.Next()
				return
			}
		}

		// 2. 如果路徑需要保護，則執行 gorilla/csrf 的邏輯
		// 建立一個假的 http.Handler 來呼叫 c.Next()
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})
		// 將 Gin 的上下文 c.Writer 和 c.Request 傳遞給 gorilla/csrf 的保護器
		csrfProtector(nextHandler).ServeHTTP(c.Writer, c.Request)

		if c.Writer.Status() == http.StatusForbidden {
			log.Info("csrf failed")
			c.Abort()
			return
		}
	}
}
