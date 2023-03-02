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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	csicommon "github.com/cubefs/cubefs-csi/pkg/csi-common"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	KVolumeName   = "volName"
	KMasterAddr   = "masterAddr"
	KLogLevel     = "logLevel"
	KLogDir       = "logDir"
	KOwner        = "owner"
	KMountPoint   = "mountPoint"
	KExporterPort = "exporterPort"
	KProfPort     = "profPort"
	KCrossZone    = "crossZone"
	KEnableToken  = "enableToken"
	KZoneName     = "zoneName"
	KConsulAddr   = "consulAddr"
	KVolType      = "volType"
)

const (
	defaultClientConfPath     = "/cfs/conf/"
	defaultLogDir             = "/cfs/logs/"
	defaultExporterPort   int = 9513
	defaultProfPort       int = 10094
	defaultLogLevel           = "info"
	jsonFileSuffix            = ".json"
	defaultConsulAddr         = "http://consul-service.cubefs.svc.cluster.local:8500"
	defaultVolType            = "0"
)

const {
	ErrCodeVolNotExists = 7
	ErrCodeDuplicateVol = 12
}

type cfsServer struct {
	clientConfFile string
	masterAddrs    []string
	clientConf     map[string]string
}

// Create and Delete Volume Response
type cfsServerResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data,omitempty"`
}

func newCfsServer(volName string, param map[string]string) (cs *cfsServer, err error) {
	masterAddr := param[KMasterAddr]
	if len(volName) == 0 || len(masterAddr) == 0 {
		return nil, fmt.Errorf("invalid argument for initializing cfsServer")
	}

	newVolName := getValueWithDefault(param, KVolumeName, volName)
	clientConfFile := defaultClientConfPath + newVolName + jsonFileSuffix
	newOwner := csicommon.ShortenString(fmt.Sprintf("csi_%d", time.Now().UnixNano()), 20)
	param[KMasterAddr] = masterAddr
	param[KVolumeName] = newVolName
	param[KOwner] = getValueWithDefault(param, KOwner, newOwner)
	param[KLogLevel] = getValueWithDefault(param, KLogLevel, defaultLogLevel)
	param[KLogDir] = defaultLogDir + newVolName
	param[KConsulAddr] = getValueWithDefault(param, KConsulAddr, defaultConsulAddr)
	param[KVolType] = getValueWithDefault(param, KVolType,  defaultVolType)
	return &cfsServer{
		clientConfFile: clientConfFile,
		masterAddrs:    strings.Split(masterAddr, ","),
		clientConf:     param,
	}, err
}

func getValueWithDefault(param map[string]string, key string, defaultValue string) string {
	value := param[key]
	if len(value) == 0 {
		value = defaultValue
	}

	return value
}

func (cs *cfsServer) persistClientConf(mountPoint string) error {
	exporterPort, _ := getFreePort(defaultExporterPort)
	profPort, _ := getFreePort(defaultProfPort)
	cs.clientConf[KMountPoint] = mountPoint
	cs.clientConf[KExporterPort] = strconv.Itoa(exporterPort)
	cs.clientConf[KProfPort] = strconv.Itoa(profPort)
	_ = os.Mkdir(cs.clientConf[KLogDir], 0777)
	clientConfBytes, _ := json.Marshal(cs.clientConf)
	err := ioutil.WriteFile(cs.clientConfFile, clientConfBytes, 0444)
	if err != nil {
		return status.Errorf(codes.Internal, "create client config file fail. err: %v", err.Error())
	}

	glog.V(0).Infof("create client config file success, volumeId:%v", cs.clientConf[KVolumeName])
	return nil
}

func (cs *cfsServer) createVolume(capacityGB int64) (err error) {
	valName := cs.clientConf[KVolumeName]
	owner := cs.clientConf[KOwner]
	crossZone := cs.clientConf[KCrossZone]
	token := cs.clientConf[KEnableToken]
	zone := cs.clientConf[KZoneName]
	volType := cs.clientConf[KVolType]

	return cs.forEachMasterAddr("CreateVolume", func(addr string) error {
		url := fmt.Sprintf("http://%s/admin/createVol?name=%s&capacity=%v&owner=%v&crossZone=%v&enableToken=%v&zoneName=%v&volType=%v",
			addr, valName, capacityGB, owner, crossZone, token, zone, volType)
		glog.Infof("createVol url: %v", url)
		resp, err := cs.executeRequest(url)
		if err != nil {
			return err
		}

		if resp.Code != 0 {
			if resp.Code == ErrCodeDuplicateVol {
				glog.Warningf("duplicate to create volume. url(%v) code=%v msg: %v", url, ErrCodeDuplicateVol, resp.Msg)
				return nil
			}

			return fmt.Errorf("create volume failed: url(%v) code=(%v), msg: %v", url, resp.Code, resp.Msg)
		}

		return nil
	})
}

func (cs *cfsServer) forEachMasterAddr(stage string, f func(addr string) error) (err error) {
	for _, addr := range cs.masterAddrs {
		if err = f(addr); err == nil {
			break
		}

		glog.Warningf("try %s with master %q failed: %v", stage, addr, err)
	}

	if err != nil {
		glog.Errorf("%s failed with all masters: %v", stage, err)
		return err
	}

	return nil
}

func (cs *cfsServer) deleteVolume() (err error) {
	ownerMd5, err := cs.getOwnerMd5()
	if err != nil {
		return err
	}

	valName := cs.clientConf[KVolumeName]
	return cs.forEachMasterAddr("DeleteVolume", func(addr string) error {
		url := fmt.Sprintf("http://%s/vol/delete?name=%s&authKey=%v", addr, valName, ownerMd5)
		glog.Infof("deleteVol url: %v", url)
		resp, err := cs.executeRequest(url)
		if err != nil {
			return err
		}

		if resp.Code != 0 {
			if resp.Code == ErrCodeVolNotExists {
				glog.Warningf("volume[%s] not exists, assuming the volume has already been deleted. code:%v, msg:%v",
					valName, resp.Code, resp.Msg)
				return nil
			}
			return fmt.Errorf("delete volume[%s] is failed. code:%v, msg:%v", valName, resp.Code, resp.Msg)
		}

		return nil
	})
}

func (cs *cfsServer) executeRequest(url string) (*cfsServerResponse, error) {
	httpResp, err := http.Get(url)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "request url failed, url(%v) err(%v)", url, err)
	}

	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "read http response body, url(%v) bodyLen(%v) err(%v)", url, len(body), err)
	}

	resp := &cfsServerResponse{}
	if err := json.Unmarshal(body, resp); err != nil {
		return nil, status.Errorf(codes.Unavailable, "unmarshal http response body, url(%v) msg(%v) err(%v)", url, resp.Msg, err)
	}
	return resp, nil
}

func (cs *cfsServer) runClient() error {
	return mountVolume(cs.clientConfFile)
}

func (cs *cfsServer) expandVolume(capacityGB int64) (err error) {
	ownerMd5, err := cs.getOwnerMd5()
	if err != nil {
		return err
	}

	volName := cs.clientConf[KVolumeName]

	return cs.forEachMasterAddr("ExpandVolume", func(addr string) error {
		url := fmt.Sprintf("http://%s/vol/expand?name=%s&authKey=%v&capacity=%v", addr, volName, ownerMd5, capacityGB)
		glog.Infof("expandVolume url: %v", url)
		resp, err := cs.executeRequest(url)
		if err != nil {
			return err
		}

		if resp.Code != 0 {
			return fmt.Errorf("expand volume[%v] failed, code:%v, msg:%v", volName, resp.Code, resp.Msg)
		}

		return nil
	})
}

func (cs *cfsServer) getOwnerMd5() (string, error) {
	owner := cs.clientConf[KOwner]
	key := md5.New()
	if _, err := key.Write([]byte(owner)); err != nil {
		return "", status.Errorf(codes.Internal, "calc owner[%v] md5 fail. err(%v)", owner, err)
	}

	return hex.EncodeToString(key.Sum(nil)), nil
}
