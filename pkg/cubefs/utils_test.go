/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cubefs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantProt string
		wantAddr string
		wantErr  bool
	}{
		{
			// Test case for a valid Unix socket endpoint with correct protocol and path
			name:     "valid unix endpoint",
			endpoint: "unix:///tmp/test.sock",
			wantProt: "unix",
			wantAddr: "/tmp/test.sock",
			wantErr:  false,
		},
		{
			// Test case for a valid TCP endpoint with correct protocol, IP and port
			name:     "valid tcp endpoint",
			endpoint: "tcp://127.0.0.1:8080",
			wantProt: "tcp",
			wantAddr: "127.0.0.1:8080",
			wantErr:  false,
		},
		{
			// Test case for handling empty endpoint string
			name:     "invalid empty endpoint",
			endpoint: "",
			wantProt: "",
			wantAddr: "",
			wantErr:  true,
		},
		{
			// Test case for handling invalid endpoint format
			name:     "invalid format endpoint",
			endpoint: "invalid://addr",
			wantProt: "",
			wantAddr: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prot, addr, err := parseEndpoint(tt.endpoint)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProt, prot)
				assert.Equal(t, tt.wantAddr, addr)
			}
		})
	}
}
func TestGetFreePort(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			// Test case for successfully getting an available port
			name:    "get free port success",
			wantErr: false,
		},
	}

	defaultPort := 8080
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := getFreePort(defaultPort)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, defaultPort, port)
			} else {
				assert.NoError(t, err)
				assert.Greater(t, port, 0)
				assert.Less(t, port, 65536)
			}
		})
	}
}
func TestExecCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			// Test basic command execution with echo
			name:    "valid command",
			command: "echo",
			args:    []string{"test"},
			wantErr: false,
		},
		{
			// Test behavior with a non-existent command
			name:    "invalid command",
			command: "invalidcommand",
			args:    []string{},
			wantErr: true,
		},
		{
			// Test behavior with empty command string
			name:    "empty command",
			command: "",
			args:    []string{},
			wantErr: true,
		},
		{
			// Test command execution with multiple arguments
			name:    "command with multiple args",
			command: "ls",
			args:    []string{"-l", "-a"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := execCommand(tt.command, tt.args...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, out)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, out)
			}
		})
	}
}

func TestPathExists(t *testing.T) {
	t.Run("path exists", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "exists-test")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		exists, err := pathExists(tmpFile.Name())
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("path does not exist", func(t *testing.T) {
		exists, err := pathExists("/nonexistent/path")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}
