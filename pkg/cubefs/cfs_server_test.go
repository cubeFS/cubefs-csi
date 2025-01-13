package cubefs

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestNewCfsServer(t *testing.T) {
	t.Run("valid parameters", func(t *testing.T) {
		param := map[string]string{
			KMasterAddr: "127.0.0.1",
			KVolumeName: "testVol",
		}
		server, err := newCfsServer("testVol", param)
		assert.Nil(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "testVol", server.clientConf[KVolumeName])
	})

	t.Run("missing volume name", func(t *testing.T) {
		param := map[string]string{
			KMasterAddr: "127.0.0.1",
		}
		server, err := newCfsServer("", param)
		assert.Nil(t, server)
		assert.NotNil(t, err)
	})

	t.Run("missing master address", func(t *testing.T) {
		param := map[string]string{
			KVolumeName: "testVol",
		}
		server, err := newCfsServer("testVol", param)
		assert.Nil(t, server)
		assert.NotNil(t, err)
	})
}

func TestGetOwnerMd5(t *testing.T) {
	param := map[string]string{
		KMasterAddr: "127.0.0.1",
		KVolumeName: "testVol",
	}
	server, _ := newCfsServer("testVol", param)
	val, err := server.getOwnerMd5()
	assert.NotEmpty(t, val)
	assert.Nil(t, err)
}

func TestGetRequest(t *testing.T) {
	monkey.Patch(http.Get, func(url string) (resp *http.Response, err error) {
		return &http.Response{}, nil
	})
	monkey.Patch(io.ReadAll, func(r io.Reader) (resp []byte, err error) {
		rsp := cfsServerResponse{
			Code: 200,
			Msg:  "OK",
			Data: "Good",
		}
		return json.Marshal(rsp)
	})
	defer monkey.UnpatchAll()

	s := &cfsServer{
		clientConf: map[string]string{"dummyKey": "dummyValue"},
	}
	resp, err := s.executeRequest("http://example.com/")
	assert.NoError(t, err)
	assert.Equal(t, resp.Code, 200)
	assert.Equal(t, resp.Msg, "OK")
	assert.Equal(t, resp.Data, "Good")
}

func TestPersistClientConf(t *testing.T) {
	monkey.Patch(os.WriteFile, func(filename string, data []byte, perm fs.FileMode) (err error) {
		return nil
	})
	defer monkey.UnpatchAll()
	s := &cfsServer{
		clientConf: map[string]string{"mockKey": "mockValue"},
	}
	err := s.persistClientConf("")
	assert.NoError(t, err)
}

func TestGetValueWithDefault(t *testing.T) {
	param := map[string]string{
		"someKey": "someValue",
	}
	got := getValueWithDefault(param, "someKey", "defaultVal")
	assert.Equal(t, "someValue", got)

	got = getValueWithDefault(param, "missingKey", "defaultVal")
	assert.Equal(t, "defaultVal", got)
}
