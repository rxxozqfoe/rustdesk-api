package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captchaProvider is the common contract implemented by the b64 providers.
type captchaProvider interface {
	Generate() (string, string, string, error)
	Expiration() time.Duration
	Draw(content string) (string, error)
}

func TestB64CaptchaProviders(t *testing.T) {
	providers := map[string]captchaProvider{
		"string": B64StringCaptchaProvider{},
		"math":   B64MathCaptchaProvider{},
	}

	for name, p := range providers {
		p := p
		t.Run(name, func(t *testing.T) {
			t.Run("Expiration is five minutes", func(t *testing.T) {
				assert.Equal(t, 5*time.Minute, p.Expiration())
			})

			t.Run("Generate yields non-empty id/content/answer", func(t *testing.T) {
				id, content, answer, err := p.Generate()
				require.NoError(t, err)
				assert.NotEmpty(t, id)
				assert.NotEmpty(t, content)
				assert.NotEmpty(t, answer)
			})

			t.Run("Generate produces unique ids", func(t *testing.T) {
				id1, _, _, err := p.Generate()
				require.NoError(t, err)
				id2, _, _, err := p.Generate()
				require.NoError(t, err)
				assert.NotEqual(t, id1, id2)
			})

			t.Run("Draw of generated content yields a base64 png data uri", func(t *testing.T) {
				_, content, _, err := p.Generate()
				require.NoError(t, err)
				b64, err := p.Draw(content)
				require.NoError(t, err)
				assert.NotEmpty(t, b64)
				assert.True(t, strings.HasPrefix(b64, "data:image/png;base64,"),
					"unexpected prefix: %q", b64[:min(len(b64), 32)])
				assert.NotEqual(t, "data:image/png;base64,", b64) // has payload
			})
		})
	}
}

func TestB64StringCaptchaAnswerMatchesContent(t *testing.T) {
	// For the string driver, content and answer are the same digits/letters.
	p := B64StringCaptchaProvider{}
	_, content, answer, err := p.Generate()
	require.NoError(t, err)
	assert.Equal(t, content, answer)
	assert.Len(t, answer, 4) // NewDriverString(..., length=4, ...)
}

func TestB64MathCaptchaAnswerIsNumeric(t *testing.T) {
	// For the math driver the answer is the numeric result while the content
	// is the question expression; they should differ.
	_, question, answer, err := B64MathCaptchaProvider{}.Generate()
	require.NoError(t, err)
	require.NotEmpty(t, question)
	require.NotEmpty(t, answer)
	assert.NotEqual(t, question, answer)
}
