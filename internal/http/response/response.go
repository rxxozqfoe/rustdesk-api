package response

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"net/http"
)

// localizerFn and logger are wired once at startup via SetLocalizer / SetLogger
// (called from cmd/apimain.go's composition root). This is the one permitted
// package-level state in this package: func pointers are set exactly once before
// any handler runs, never mutated, and never read from the request path otherwise.
var (
	localizerFn func(lang string) *i18n.Localizer
	logger      *logrus.Logger
)

// SetLocalizer wires the i18n localizer constructor. Call once at startup.
func SetLocalizer(fn func(lang string) *i18n.Localizer) { localizerFn = fn }

// SetLogger wires the logger used when translation fails. Call once at startup.
func SetLogger(l *logrus.Logger) { logger = l }

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
type PageData struct {
	Page  int         `json:"page"`
	Total int         `json:"total"`
	List  interface{} `json:"list"`
}

type DataResponse struct {
	Total uint        `json:"total"`
	Data  interface{} `json:"data"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func SendResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		code, message, data,
	})
}

func Success(c *gin.Context, data interface{}) {
	SendResponse(c, 0, "success", data)
}

func Fail(c *gin.Context, code int, message string) {
	SendResponse(c, code, message, nil)
}

func Error(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: message,
	})
}

type ServerConfigResponse struct {
	IdServer    string `json:"id_server"`
	Key         string `json:"key"`
	RelayServer string `json:"relay_server"`
	ApiServer   string `json:"api_server"`
}

func TranslateMsg(c *gin.Context, messageId string) string {
	localizer := localizerFn(c.GetHeader("Accept-Language"))
	errMsg, err := localizer.LocalizeMessage(&i18n.Message{
		ID: messageId,
	})
	if err != nil {
		logger.Warn("LocalizeMessage Error: " + err.Error())
		errMsg = messageId
	}
	return errMsg
}
func TranslateTempMsg(c *gin.Context, messageId string, templateData map[string]interface{}) string {
	localizer := localizerFn(c.GetHeader("Accept-Language"))
	errMsg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageId,
		},
		TemplateData: templateData,
	})
	if err != nil {
		logger.Warn("LocalizeMessage Error: " + err.Error())
		errMsg = messageId
	}
	return errMsg
}
func TranslateParamMsg(c *gin.Context, messageId string, params ...string) string {
	localizer := localizerFn(c.GetHeader("Accept-Language"))
	templateData := make(map[string]interface{})
	for i, v := range params {
		k := fmt.Sprintf("P%d", i)
		templateData[k] = v
	}
	errMsg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageId,
		},
		TemplateData: templateData,
	})
	if err != nil {
		logger.Warn("LocalizeMessage Error: " + err.Error())
		errMsg = messageId
	}
	return errMsg
}
