package ezapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invopop/ctxi18n"
)

func I18n(defaultLanguage string, isLocalExist func(lang string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Param("lang")

		if lang != "" {
			ctx, err := ctxi18n.WithLocale(c.Request.Context(), strings.ToLower(lang))
			if err != nil {
				c.Writer.WriteHeader(http.StatusInternalServerError)
				c.Writer.Write([]byte("Failed to set locale: " + err.Error()))
				c.Abort()
				return
			}
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}
