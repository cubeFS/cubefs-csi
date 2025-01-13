package cubefs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewController(t *testing.T) {
	driver, _ := NewMockupDriver()
	svc := NewControllerServer(driver)
	assert.NotNil(t, svc)
}
