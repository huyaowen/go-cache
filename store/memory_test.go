package store

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestApi(t *testing.T) {
	m := Backend(MEMORY)
	m.Set("test", "test", 5*time.Second)
	time.Sleep(2 * time.Second)
	r, _ := m.Get("test")
	assert.Equal(t, r, "test")
	t.Log(r)

}
