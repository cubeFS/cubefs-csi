package cubefs

import (
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func NewMockupDriver() (*driver, error) {
	monkey.Patch(os.ReadFile, func(filename string) ([]byte, error) {
		if filename == "/var/run/secrets/kubernetes.io/serviceaccount/token" {
			return []byte("mock-token-content"), nil
		}
		return []byte{}, os.ErrNotExist
	})
	defer monkey.UnpatchAll()

	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	os.Setenv("KUBERNETES_SERVICE_PORT", "6443")
	defer func() {
		os.Setenv("KUBERNETES_SERVICE_HOST", host)
		os.Setenv("KUBERNETES_SERVICE_PORT", port)
	}()

	driver, err := NewDriver(Config{
		NodeID:     "test-node",
		DriverName: "test-driver",
		Version:    "test-version",
	})
	return driver, err
}

func TestNewDriver(t *testing.T) {
	_, err := NewMockupDriver()
	assert.NoError(t, err)
}
