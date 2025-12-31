package ezapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invopop/ctxi18n"
)

func I18n(defaultLanguage string, isLocalExist func(lang string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var lang string
		switch {
		case c.GetString("line.liff.locale") != "":
			lang = c.GetString("line.liff.locale")
		case c.Param("lang") != "":
			lang = c.Param("lang")
		default:
			lang = defaultLanguage
		}

		if lang != "" {
			if !isLocalExist(lang) {
				c.Next()
				return
			}
			ctx, err := ctxi18n.WithLocale(c.Request.Context(), strings.ToLower(lang))
			if err != nil {
				c.Writer.WriteHeader(http.StatusInternalServerError)
				c.Writer.Write([]byte("Failed to set locale: " + err.Error()))
				c.Abort()
				return
			}
			c.Request = c.Request.Clone(ctx)
		}

		c.Next()
	}
}
