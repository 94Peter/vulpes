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
		}

		if lang != "" {
			if !isLocalExist(lang) {
				lang = defaultLanguage
			}
			ctx, err := ctxi18n.WithLocale(c.Request.Context(), strings.ToLower(lang))
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to set locale: "+err.Error())
				c.Abort()
				return
			}
			c.Request = c.Request.Clone(ctx)
		}

		c.Next()
	}
}
