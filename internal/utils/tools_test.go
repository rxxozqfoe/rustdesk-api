package utils

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMd5(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"known hello", "hello", "5d41402abc4b2a76b9719d911017c592"},
		{"empty string", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"rustdesk salt", "secret" + "rustdesk-api", Md5("secret" + "rustdesk-api")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Md5(tt.in)
			assert.Equal(t, tt.want, got)
			// md5 hex output is always 32 chars
			assert.Len(t, got, 32)
		})
	}

	t.Run("deterministic", func(t *testing.T) {
		assert.Equal(t, Md5("abc"), Md5("abc"))
		assert.NotEqual(t, Md5("abc"), Md5("abd"))
	})
}

func TestCopyStructByJson(t *testing.T) {
	type src struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type dst struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	t.Run("copies matching fields", func(t *testing.T) {
		s := src{Name: "alice", Age: 30}
		var d dst
		CopyStructByJson(s, &d)
		assert.Equal(t, "alice", d.Name)
		assert.Equal(t, 30, d.Age)
	})

	t.Run("zero value source", func(t *testing.T) {
		var s src
		var d dst
		CopyStructByJson(s, &d)
		assert.Equal(t, "", d.Name)
		assert.Equal(t, 0, d.Age)
	})

	t.Run("non-pointer dst is a no-op", func(t *testing.T) {
		s := src{Name: "bob", Age: 1}
		var d dst
		// Unmarshal into a non-pointer fails internally and is swallowed.
		CopyStructByJson(s, d)
		assert.Equal(t, "", d.Name)
	})
}

func TestCopyStructToMap(t *testing.T) {
	type src struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	t.Run("struct to map", func(t *testing.T) {
		m := CopyStructToMap(src{Name: "alice", Age: 30})
		require.NotNil(t, m)
		assert.Equal(t, "alice", m["name"])
		// JSON numbers decode to float64
		assert.Equal(t, float64(30), m["age"])
	})

	t.Run("empty struct", func(t *testing.T) {
		m := CopyStructToMap(struct{}{})
		require.NotNil(t, m)
		assert.Empty(t, m)
	})

	t.Run("non-object marshals to non-map -> nil", func(t *testing.T) {
		// A plain string marshals to a JSON string, which cannot unmarshal
		// into map[string]interface{}, so nil is returned.
		assert.Nil(t, CopyStructToMap("just a string"))
	})
}

func TestSafeGo(t *testing.T) {
	t.Run("runs function with params", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		var got int
		SafeGo(func(a, b int) {
			defer wg.Done()
			got = a + b
		}, 2, 3)
		wg.Wait()
		assert.Equal(t, 5, got)
	})

	t.Run("recovers from panic without crashing", func(t *testing.T) {
		done := make(chan struct{})
		SafeGo(func() {
			defer close(done)
			panic("boom")
		})
		// If the panic were not recovered the test process would crash.
		<-done
	})

	t.Run("non-function value returns without panic", func(t *testing.T) {
		// Should not panic; just logs and returns. Nothing to wait on,
		// but the call itself must not crash the test.
		assert.NotPanics(t, func() {
			SafeGo(42)
		})
	})
}

func TestRandomString(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	tests := []struct {
		name string
		n    int
	}{
		{"zero length", 0},
		{"length one", 1},
		{"length sixteen", 16},
		{"length 256", 256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := RandomString(tt.n)
			assert.Len(t, s, tt.n)
			for _, c := range s {
				assert.Contains(t, charset, string(c),
					"character %q outside expected charset", string(c))
			}
		})
	}

	t.Run("randomness across calls", func(t *testing.T) {
		// Two long random strings should be different with overwhelming
		// probability.
		assert.NotEqual(t, RandomString(64), RandomString(64))
	})
}

func TestKeys(t *testing.T) {
	t.Run("string keys", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		got := Keys(m)
		sort.Strings(got)
		assert.Equal(t, []string{"a", "b", "c"}, got)
	})

	t.Run("empty map", func(t *testing.T) {
		got := Keys(map[string]int{})
		assert.Empty(t, got)
		assert.NotNil(t, got)
	})

	t.Run("nil map", func(t *testing.T) {
		var m map[int]string
		got := Keys(m)
		assert.Empty(t, got)
	})
}

func TestValues(t *testing.T) {
	t.Run("int values", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2, "c": 3}
		got := Values(m)
		sort.Ints(got)
		assert.Equal(t, []int{1, 2, 3}, got)
	})

	t.Run("empty map", func(t *testing.T) {
		got := Values(map[string]int{})
		assert.Empty(t, got)
		assert.NotNil(t, got)
	})

	t.Run("nil map", func(t *testing.T) {
		var m map[int]string
		got := Values(m)
		assert.Empty(t, got)
	})
}

func TestInArray(t *testing.T) {
	tests := []struct {
		name string
		k    string
		arr  []string
		want bool
	}{
		{"present", "b", []string{"a", "b", "c"}, true},
		{"first element", "a", []string{"a", "b", "c"}, true},
		{"last element", "c", []string{"a", "b", "c"}, true},
		{"absent", "z", []string{"a", "b", "c"}, false},
		{"empty slice", "a", []string{}, false},
		{"nil slice", "a", nil, false},
		{"empty needle present", "", []string{"", "a"}, true},
		{"empty needle absent", "", []string{"a"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, InArray(tt.k, tt.arr))
		})
	}
}

func TestStringConcat(t *testing.T) {
	tests := []struct {
		name string
		strs []string
		want string
	}{
		{"no args", nil, ""},
		{"single", []string{"a"}, "a"},
		{"multiple", []string{"a", "b", "c"}, "abc"},
		{"with empties", []string{"a", "", "b"}, "ab"},
		{"all empty", []string{"", "", ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StringConcat(tt.strs...))
		})
	}
}
