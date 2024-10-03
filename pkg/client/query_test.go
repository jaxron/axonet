package client_test

import (
	"testing"

	"github.com/jaxron/axonet/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	t.Parallel()

	t.Run("Get", func(t *testing.T) {
		t.Parallel()

		q := client.Query{
			"foo": []string{"bar"},
			"baz": []string{"qux", "quux"},
		}

		assert.Equal(t, "bar", q.Get("foo"))
		assert.Equal(t, "qux", q.Get("baz"))
		assert.Equal(t, "", q.Get("nonexistent"))
	})

	t.Run("Set", func(t *testing.T) {
		t.Parallel()

		q := client.Query{}

		q.Set("foo", "bar")
		assert.Equal(t, []string{"bar"}, q["foo"])

		q.Set("foo", "baz")
		assert.Equal(t, []string{"baz"}, q["foo"])

		q.Set("empty", "")
		_, exists := q["empty"]
		assert.False(t, exists)
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		q := client.Query{}

		q.Add("foo", "bar")
		assert.Equal(t, []string{"bar"}, q["foo"])

		q.Add("foo", "baz")
		assert.Equal(t, []string{"bar", "baz"}, q["foo"])

		q.Add("empty", "")
		_, exists := q["empty"]
		assert.False(t, exists)
	})

	t.Run("Encode", func(t *testing.T) {
		t.Parallel()

		q := client.Query{
			"foo":     []string{"bar"},
			"baz":     []string{"qux", "quux"},
			"empty":   []string{},
			"space":   []string{"hello world"},
			"special": []string{"!@#$%^&*()"},
		}

		encoded := q.Encode()
		assert.Contains(t, encoded, "foo=bar")
		assert.Contains(t, encoded, "baz=qux")
		assert.Contains(t, encoded, "baz=quux")
		assert.NotContains(t, encoded, "empty=")
		assert.Contains(t, encoded, "space=hello+world")
		assert.Contains(t, encoded, "special=%21%40%23%24%25%5E%26%2A%28%29")

		// Check that the keys are sorted
		assert.True(t, len(encoded) > 0 && encoded[0] == 'b')
	})

	t.Run("Encode empty query", func(t *testing.T) {
		t.Parallel()

		q := client.Query{}
		assert.Equal(t, "", q.Encode())
	})
}
