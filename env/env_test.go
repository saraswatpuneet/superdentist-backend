package env

import (
	"bufio"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	NotFoundKeyName = "NotFound"
	FoundKeyName    = "Found"
	FloatKeyName    = "float64"
)

func TestFindKeyValue(t *testing.T) {
	k, v := findKeyValue(" test= value")
	assert.Equal(t, "test", k)
	assert.Equal(t, "value", v)

	k, v = findKeyValue("\ttest=\tvalue\t\n")
	assert.Equal(t, "test", k)
	assert.Equal(t, "value", v)
}

func TestLoad(t *testing.T) {
	filename := "/tmp/.env"
	// plain value without quote
	if dotenv, err := os.Create(filename); err == nil {
		defer dotenv.Close()
		defer os.Remove(filename)
		buffer := bufio.NewWriter(dotenv)
		buffer.WriteString("app=myapp\n")
		buffer.WriteString("export exportation=myexports")
		buffer.Flush()

		err := Set("root", "/tmp")
		assert.NoError(t, err, "Fail in Set")

		Load("/tmp/.env")

		assert.Equal(t, String("app"), "myapp")
		assert.Equal(t, String("exportation"), "myexports")
	}
}

func TestString(t *testing.T) {
	assert.Equal(t, String(NotFoundKeyName), "")
	err := Set(FoundKeyName, "something")
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, String(FoundKeyName), "something")
	assert.Equal(t, String(NotFoundKeyName, "default"), "default")
}

func TestString2(t *testing.T) {
	err := Set(FoundKeyName, "something@dev")
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, String(FoundKeyName), "something@dev")
}

func TestStrings(t *testing.T) {
	err := Set("StringList", "a,b,c")
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, Strings("StringList"), []string{"a", "b", "c"})
	assert.Equal(t, Strings(NotFoundKeyName, []string{"a", "b", "c"}), []string{"a", "b", "c"})
}

func TestInt(t *testing.T) {
	err := Set("integer", 123)
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, Int("integer"), 123)
	assert.Equal(t, Int(NotFoundKeyName, 123), 123)
}

func TestInt64(t *testing.T) {
	err := Set("int64", int64(123))
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, int64(123), Int64("int64"))
	assert.Equal(t, int64(123), Int64(NotFoundKeyName, 123))
}

func TestUint(t *testing.T) {
	err := Set("uint", uint(123))
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, uint(123), Uint("uint"))
	assert.Equal(t, uint(123), Uint(NotFoundKeyName, 123))
}

func TestUint64(t *testing.T) {
	err := Set("uint64", uint64(123))
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, uint64(123), Uint64("uint64"))
	assert.Equal(t, uint64(123), Uint64(NotFoundKeyName, 123))
}

func TestBool(t *testing.T) {
	err := Set("bool", true)
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, Bool("bool"), true)
	assert.Equal(t, Bool(NotFoundKeyName, true), true)
}

func TestFloat(t *testing.T) {
	err := Set(FloatKeyName, 12345678990.0987654321)
	assert.NoError(t, err, "Fail in Set")
	assert.Equal(t, Float(FloatKeyName), 12345678990.0987654321)
	assert.Equal(t, Float(FloatKeyName, 0.1), 12345678990.0987654321)
	assert.Equal(t, Float(NotFoundKeyName, 0.1), 0.1)
}
