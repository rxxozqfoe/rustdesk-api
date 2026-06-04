package app

import (
	"github.com/BurntSushi/toml"
	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"os"
)

// LocalizerFunc returns an i18n localizer for the given Accept-Language header
// value. Constructed by NewI18n and then stored on http/deps.HandlerDeps.
type LocalizerFunc func(lang string) *i18n.Localizer

// NewI18n loads all .toml message bundles from <ResourcesPath>/i18n and
// returns a LocalizerFunc that produces per-request localizers.
func NewI18n(cfg *config.Config) LocalizerFunc {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	dir := cfg.Gin.ResourcesPath + "/i18n"
	fileInfos, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() || fileInfo.Name()[len(fileInfo.Name())-5:] != ".toml" {
			continue
		}
		if _, err := bundle.LoadMessageFile(cfg.Gin.ResourcesPath + "/i18n/" + fileInfo.Name()); err != nil {
			panic(err)
		}
	}
	return func(lang string) *i18n.Localizer {
		if lang == "" {
			lang = cfg.Lang
		}
		if lang == "en" {
			return i18n.NewLocalizer(bundle, "en")
		}
		return i18n.NewLocalizer(bundle, lang, "en")
	}
}
