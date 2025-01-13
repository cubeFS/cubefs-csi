package cubefs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNodeServer(t *testing.T) {
	driver, _ := NewMockupDriver()
	svc := NewNodeServer(driver)
	assert.NotNil(t, svc)
}

func TestNodeGetVolumeStats(t *testing.T) {
	ctx := context.Background()
	opt, err := nodeGetVolumeStats(ctx, "/tmp")
	assert.NoError(t, err)
	assert.NotEqual(t, len(opt.Usage), 0)
	assert.NotEqual(t, opt.Usage[0].Total, 0)
}
