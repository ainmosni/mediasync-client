/*
Copyright 2020 Daniël Franke

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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ainmosni/mediasync-client/pkg/config"
)

type wp struct {
	WebPath string `json:"web_path"`
}

func getFiles(c *config.Configuration, logger *log.Logger) []wp {
	fileInfo, err := url.Parse(c.Remote)
	if err != nil {
		logger.Fatalf("Can't parse remote: %s", err)
	}

	fileInfo.Path = path.Join(fileInfo.Path, "/fileinfo")

	req, err := http.NewRequest("GET", fileInfo.String(), nil)
	if err != nil {
		logger.Fatalf("Bad request: %s", err)
	}

	req.SetBasicAuth(c.UserName, c.Password)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.Fatalf("Failed to get fileinfo: %s", err)
	}
	defer resp.Body.Close()

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, resp.Body)

	if err != nil {
		logger.Fatalf("failed to copy: %s", err)
	}

	var files []wp
	err = json.Unmarshal(buf.Bytes(), &files)
	if err != nil {
		logger.Fatalf("Couldn't parse json: %s", err)
	}
	return files
}

func findLocal(f string, c *config.Configuration) string {
	localFile := ""
	for _, p := range c.RootMapping {
		if strings.HasPrefix(f, p.RemotePath) {
			localFile = strings.ReplaceAll(f, p.RemotePath, p.LocalPath)
		}
	}
	return localFile
}

func getFile(f wp, c *config.Configuration, logger *log.Logger) {
	localFile := findLocal(f.WebPath, c)
	if localFile == "" {
		logger.Fatalf("Couldn't find config for remote file %s", f)
	}

	fileURL, err := url.Parse(c.Remote)
	if err != nil {
		logger.Fatalf("Couldn't parse remote: %s", err)
	}

	fileURL.Path = path.Join(fileURL.Path, f.WebPath)

	dir, _ := filepath.Split(localFile)
	err = os.MkdirAll(dir, 0775)
	if err != nil {
		logger.Fatalf("Couldn't create dir: %s", err)
	}
	output, err := os.Create(localFile)
	if err != nil {
		logger.Fatalf("Couldn't create file: %s", err)
	}
	defer output.Close()

	req, err := http.NewRequest("GET", fileURL.String(), nil)
	if err != nil {
		logger.Fatalf("Bad request: %s", err)
	}
	req.SetBasicAuth(c.UserName, c.Password)

	start := time.Now()
	logger.Printf("Downloading %s to %s.", fileURL.String(), localFile)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Fatalf("Couldn't download %s: %s", fileURL.String(), err)
	}
	defer resp.Body.Close()

	written, err := io.Copy(output, resp.Body)
	if err != nil {
		logger.Fatalf("Failed downloading %s: %s", fileURL, err)
	}
	end := time.Since(start)
	secs := end.Seconds()
	bps := float64(written) / secs

	logger.Printf("Finished downloading %s in %f seconds (%f bps)", localFile, secs, bps)

	delReq, err := http.NewRequest("DELETE", fileURL.String(), nil)
	if err != nil {
		logger.Fatalf("Bad request: %s", err)
	}
	delReq.SetBasicAuth(c.UserName, c.Password)

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		logger.Fatalf("Failed to delete %s: %s", fileURL.String(), err)
	}
	delResp.Body.Close()
}

func main() {
	logger, err := syslog.NewLogger(syslog.LOG_INFO, 0)
	if err != nil {
		panic(fmt.Errorf("can't init logger: %s", err))
	}
	c, err := config.GetConfig()
	if err != nil {
		logger.Fatalf("Can't get configuration: %s", err)
	}

	files := getFiles(c, logger)

	for _, f := range files {
		getFile(f, c, logger)
	}
}
