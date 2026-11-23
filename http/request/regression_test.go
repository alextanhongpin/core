package request_test

import (
	"bytes"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/request"
	"github.com/stretchr/testify/assert"
)

func TestDecodeJSONEmptyBodyStillValidates(t *testing.T) {
	// Regression: an empty body returned nil before Validate ran, so
	// clients could bypass validation by sending nothing.
	r := httptest.NewRequest("POST", "/login", nil)
	var req loginRequest
	err := request.DecodeJSON(r, &req)

	var ve interface{ Map() map[string][]string }
	assert.ErrorAs(t, err, &ve)
}

func TestParse(t *testing.T) {
	// Regression: Parse replaced the body with an already-drained reader and
	// read without a size limit.
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"a@b.c"}`))
	req, err := request.Parse[loginRequest](r)
	assert.NoError(t, err)
	assert.Equal(t, "a@b.c", req.Email)

	b, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"email":"a@b.c"}`, string(b))

	r = httptest.NewRequest("POST", "/", strings.NewReader(`{"email":"a@b.c"}`))
	_, err = request.Parse[loginRequest](r, request.WithMaxBodySize(4))
	assert.ErrorIs(t, err, request.ErrBodyTooLarge)
}

func TestReadIsBounded(t *testing.T) {
	body := bytes.Repeat([]byte("a"), request.MaxBodySize+1)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))

	_, err := request.Read(r)
	assert.ErrorIs(t, err, request.ErrBodyTooLarge)

	// The body is not lost.
	got, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	assert.Equal(t, len(body), len(got))
}

func TestDecodeQueryRejectsOverflow(t *testing.T) {
	// Regression: values were parsed as 64-bit and silently wrapped into
	// narrower fields.
	var q struct {
		Small int8    `form:"small"`
		U     uint8   `form:"u"`
		F     float32 `form:"f"`
	}

	for _, query := range []string{"small=200", "u=256", "f=1e40"} {
		r := httptest.NewRequest("GET", "/?"+query, nil)
		err := request.DecodeQuery(r, &q)
		assert.Error(t, err, query)
	}

	r := httptest.NewRequest("GET", "/?small=-128&u=255&f=1.5", nil)
	assert.NoError(t, request.DecodeQuery(r, &q))
	assert.Equal(t, int8(-128), q.Small)
	assert.Equal(t, uint8(255), q.U)
	assert.Equal(t, float32(1.5), q.F)
}

func TestValueRangeRejectsInvalidInput(t *testing.T) {
	// Regression: unparsable input became 0 (or NaN) and passed range checks.
	_, err := request.Value("abc").IntRange(0, 10)
	assert.Error(t, err)
	_, err = request.Value("abc").Int64Range(0, 10)
	assert.Error(t, err)
	_, err = request.Value("abc").Int32Range(0, 10)
	assert.Error(t, err)
	_, err = request.Value("NaN").Float64Range(0, 10)
	assert.Error(t, err)
	_, err = request.Value("99999999999").Int32Range(-1<<31, 1<<31-1)
	assert.Error(t, err)

	assert.Equal(t, 5, request.Value("abc").IntRangeOr(0, 10, 5))
	assert.Equal(t, 3.0, request.Value("Inf").Float64Or(3))
}

func TestValueOrKeepsExplicitZero(t *testing.T) {
	// Regression: IntOr used cmp.Or, so "?offset=0" returned the default.
	assert.Equal(t, 0, request.Value("0").IntOr(10))
	assert.Equal(t, int64(0), request.Value("0").Int64Or(10))
	assert.Equal(t, int32(0), request.Value("0").Int32Or(10))
	assert.Equal(t, 10, request.Value("").IntOr(10))
	assert.Equal(t, 10, request.Value("x").IntOr(10))
	assert.True(t, request.Value("maybe").BoolOr(true))
}

