package app

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/fr"
	"github.com/go-playground/locales/ko"
	"github.com/go-playground/locales/ru"
	"github.com/go-playground/locales/zh_Hans_CN"
	"github.com/go-playground/locales/zh_Hant"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	es_translations "github.com/go-playground/validator/v10/translations/es"
	fr_translations "github.com/go-playground/validator/v10/translations/fr"
	ko_translations "github.com/go-playground/validator/v10/translations/ko"
	ru_translations "github.com/go-playground/validator/v10/translations/ru"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	zh_tw_translations "github.com/go-playground/validator/v10/translations/zh_tw"
	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"reflect"
)

// NewValidator builds an AppValidator with i18n-aware error messages for the
// seven supported languages. The resulting value is stored on
// http/deps.HandlerDeps and injected into every controller that needs it.
func NewValidator(cfg *config.Config) AppValidator {
	validate := validator.New()

	enT := en.New()
	cn := zh_Hans_CN.New()
	koT := ko.New()
	ruT := ru.New()
	esT := es.New()
	frT := fr.New()
	zhTwT := zh_Hant.New()

	uni := ut.New(enT, cn, koT, ruT, esT, frT, zhTwT)

	enTrans, _ := uni.GetTranslator("en")
	zhTrans, _ := uni.GetTranslator("zh_Hans_CN")
	koTrans, _ := uni.GetTranslator("ko")
	ruTrans, _ := uni.GetTranslator("ru")
	esTrans, _ := uni.GetTranslator("es")
	frTrans, _ := uni.GetTranslator("fr")
	zhTwTrans, _ := uni.GetTranslator("zh_Hant")

	zh_translations.RegisterDefaultTranslations(validate, zhTrans)
	en_translations.RegisterDefaultTranslations(validate, enTrans)
	ko_translations.RegisterDefaultTranslations(validate, koTrans)
	ru_translations.RegisterDefaultTranslations(validate, ruTrans)
	es_translations.RegisterDefaultTranslations(validate, esTrans)
	fr_translations.RegisterDefaultTranslations(validate, frTrans)
	zh_tw_translations.RegisterDefaultTranslations(validate, zhTwTrans)

	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		label := field.Tag.Get("label")
		if label == "" {
			return field.Name
		}
		return label
	})

	av := AppValidator{
		Validate: validate,
		UT:       uni,
		VTrans:   zhTrans,
	}

	translatorForLang := func(lang string) ut.Translator {
		switch lang {
		case "zh_CN", "zh-CN", "zh":
			trans, _ := uni.GetTranslator("zh_Hans_CN")
			return trans
		case "zh_TW", "zh-TW", "zh-tw":
			trans, _ := uni.GetTranslator("zh_Hant")
			return trans
		case "ko":
			trans, _ := uni.GetTranslator("ko")
			return trans
		case "ru":
			trans, _ := uni.GetTranslator("ru")
			return trans
		case "es":
			trans, _ := uni.GetTranslator("es")
			return trans
		case "fr":
			trans, _ := uni.GetTranslator("fr")
			return trans
		default:
			trans, _ := uni.GetTranslator("en")
			return trans
		}
	}

	av.ValidStruct = func(c *gin.Context, i interface{}) []string {
		err := validate.Struct(i)
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = cfg.Lang
		}
		trans := translatorForLang(lang)
		errList := make([]string, 0, 10)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				errList = append(errList, err.Error())
				return errList
			}
			for _, err2 := range err.(validator.ValidationErrors) {
				errList = append(errList, err2.Translate(trans))
			}
		}
		return errList
	}
	av.ValidVar = func(c *gin.Context, field interface{}, tag string) []string {
		err := validate.Var(field, tag)
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = cfg.Lang
		}
		trans := translatorForLang(lang)
		errList := make([]string, 0, 10)
		if err != nil {
			if _, ok := err.(*validator.InvalidValidationError); ok {
				errList = append(errList, err.Error())
				return errList
			}
			for _, err2 := range err.(validator.ValidationErrors) {
				errList = append(errList, err2.Translate(trans))
			}
		}
		return errList
	}
	return av
}
