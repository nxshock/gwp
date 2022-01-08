package gwp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFmtDuration(t *testing.T) {
	assert.Equal(t, "              1s", fmtDuration(time.Second))
	assert.Equal(t, "          1m  0s", fmtDuration(time.Minute))
	assert.Equal(t, "      1h      0s", fmtDuration(time.Hour))
	assert.Equal(t, "  1d          0s", fmtDuration(time.Hour*24))
	assert.Equal(t, "365d          0s", fmtDuration(time.Hour*24*365))
}
