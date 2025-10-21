package mid

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/pkg/logger"
)

var translator ut.Translator

func init() {
	//setup
	if validate, ok := binding.Validator.Engine().(*validator.Validate); ok {
		translator, _ = ut.New(en.New(), en.New()).GetTranslator("en")
		en_translations.RegisterDefaultTranslations(validate, translator)

		//using json tag names instead of field names
		validate.RegisterTagNameFunc(func(field reflect.StructField) string {
			tag := field.Tag.Get("json")
			name := strings.SplitN(tag, ",", 2)[0]

			if name == "-" {
				return ""
			}
			return name
		})
	}
}

func Error(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first.

		// Check if response already written or no errors
		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}

		//check for errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var appErr *errs.Error
			var validationErrors validator.ValidationErrors

			switch {
			case errors.As(err, &appErr):
				//app errors
				log.Error(c.Request.Context(), "error while handling request", "err", err, "fileName", appErr.FileName, "funcName", appErr.FuncName)
				//only the internal server errors need a generic err message so we do not leak any info
				if appErr.Code == http.StatusInternalServerError {
					appErr.Message = http.StatusText(http.StatusInternalServerError)
				}

				c.JSON(appErr.Code, appErr)
			case errors.As(err, &validationErrors):
				//model validation errors
				fieldErrs := make(map[string]string, len(validationErrors))
				for _, e := range validationErrors {
					fieldErrs[e.Field()] = e.Translate(translator)
				}
				err := errs.Error{
					Code:    http.StatusBadRequest,
					Message: "validation failed",
					Fields:  fieldErrs,
				}
				c.JSON(err.Code, err)
			default:
				//unknown errors
				log.Error(c.Request.Context(), "unknown error", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
			}

		}
	}
}
