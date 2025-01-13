package cubefs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIdentifierServer(t *testing.T) {
	driver, _ := NewMockupDriver()
	svc := NewIdentityServer(driver)
	assert.NotNil(t, svc)
}
