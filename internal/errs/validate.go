package errs

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var validate *validator.Validate
var translator ut.Translator

func init() {
	//setup
	validate = validator.New(validator.WithRequiredStructEnabled())
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

func Check(value any) map[string]string {
	if err := validate.Struct(value); err != nil {
		verrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return map[string]string{"err": err.Error()}
		}

		fieldErrs := make(map[string]string, len(verrors))
		for _, e := range verrors {
			fieldErrs[e.Field()] = e.Translate(translator)
		}

		return fieldErrs
	}

	return nil
}
