// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/coze-dev/cozeloop-go/internal"
)

var (
	version              = internal.Version()
	userAgentSDK         = "loop-go"
	userAgentLang        = "go"
	userAgentLangVersion = strings.TrimPrefix(runtime.Version(), "go")
	userAgentOsName      = runtime.GOOS
	userAgentOsVersion   = os.Getenv("OSVERSION")
	scene                = "loop"
	source               = "openapi"
	userAgent            = userAgentSDK + "/" + version + " " + userAgentLang + "/" + userAgentLangVersion + " " + userAgentOsName + "/" + userAgentOsVersion
	clientUserAgent      string
)

func setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Coze-Client-User-Agent", clientUserAgent)
}

func init() {
	clientUserAgent = getLoopClientUserAgent()
}

type userAgentInfo struct {
	Version     string `json:"version"`
	Lang        string `json:"lang"`
	LangVersion string `json:"lang_version"`
	OsName      string `json:"os_name"`
	OsVersion   string `json:"os_version"`
	Scene       string `json:"scene"`
	Source      string `json:"source"`
}

func getLoopClientUserAgent() string {
	data, _ := json.Marshal(userAgentInfo{
		Version:     version,
		Lang:        userAgentLang,
		LangVersion: userAgentLangVersion,
		OsName:      userAgentOsName,
		OsVersion:   userAgentOsVersion,
		Scene:       scene,
		Source:      source,
	})
	return string(data)
}
