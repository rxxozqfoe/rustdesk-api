package custom_types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoJson_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "valid json from []byte",
			input: []byte(`{"a":1}`),
			want:  `{"a":1}`,
		},
		{
			name:  "valid json from string",
			input: `[1,2,3]`,
			want:  `[1,2,3]`,
		},
		{
			name:  "empty []byte yields empty object",
			input: []byte(``),
			want:  `{}`,
		},
		{
			name:  "empty string yields empty object",
			input: ``,
			want:  `{}`,
		},
		{
			name:  "malformed json silently yields empty object (no error)",
			input: []byte(`{not json`),
			want:  `{}`,
		},
		{
			name:    "unsupported type returns error",
			input:   42,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j AutoJson
			err := j.Scan(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(j))
		})
	}
}

func TestAutoJson_Value(t *testing.T) {
	j := AutoJson(json.RawMessage(`{"k":"v"}`))
	v, err := j.Value()
	require.NoError(t, err)
	s, ok := v.(string)
	require.True(t, ok, "Value should be a string, got %T", v)
	assert.JSONEq(t, `{"k":"v"}`, s)
}

func TestAutoJson_MarshalJSON(t *testing.T) {
	j := AutoJson(json.RawMessage(`{"x":true}`))
	b, err := j.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"x":true}`, string(b))

	// AutoJson embeds in a struct and marshals transparently.
	type wrapper struct {
		Data AutoJson `json:"data"`
	}
	out, err := json.Marshal(wrapper{Data: AutoJson(json.RawMessage(`[1,2]`))})
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":[1,2]}`, string(out))
}

func TestAutoJson_UnmarshalJSON(t *testing.T) {
	var j AutoJson
	err := j.UnmarshalJSON([]byte(`{"hello":"world"}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"hello":"world"}`, string(j))

	// Round-trip: Unmarshal then Marshal must preserve content.
	b, err := j.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"hello":"world"}`, string(b))
}

func TestAutoJson_RoundTrip_ScanValue(t *testing.T) {
	original := `{"nested":{"arr":[1,2,3]},"flag":false}`
	var j AutoJson
	require.NoError(t, j.Scan([]byte(original)))

	v, err := j.Value()
	require.NoError(t, err)
	s := v.(string)

	var j2 AutoJson
	require.NoError(t, j2.Scan(s))
	assert.JSONEq(t, original, string(j2))
}

func TestAutoJson_String(t *testing.T) {
	j := AutoJson(json.RawMessage(`{"s":1}`))
	assert.JSONEq(t, `{"s":1}`, j.String())
}
