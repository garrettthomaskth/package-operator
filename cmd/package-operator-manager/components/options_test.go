package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvideOptions(t *testing.T) {
	opts, err := ProvideOptions()

	assert.Nil(t, err)
	assert.Equal(t, Options{
		MetricsAddr: ":8080",
		ProbeAddr:   ":8081",
	}, opts)
}
