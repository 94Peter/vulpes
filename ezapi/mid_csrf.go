package ezapi

import (
	"github.com/gin-gonic/gin"
)

func csrfProtectionWithExclusion(csrfMiddle gin.HandlerFunc, excludedPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 檢查目前路徑是否在排除列表中
		for _, path := range excludedPaths {
			if c.Request.URL.Path == path {
				// 如果在列表中，直接呼叫下一個處理程序，不進行 CSRF 驗證
				c.Next()
				return
			}
		}
		csrfMiddle(c)
	}
}
