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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ainmosni/mediasync-client/pkg/config"
	"github.com/ainmosni/mediasync-client/pkg/report"
	"github.com/nightlyone/lockfile"
)

const (
	lockFile = "/tmp/mediasync.lock"

	postfixLen = 8
)

type wp struct {
	WebPath string `json:"web_path"`
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", b), nil
}

func createURL(c *config.Configuration, rPath string) (*url.URL, error) {
	u, err := url.Parse(c.Remote)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, rPath)
	return u, nil
}

func reqWithAuth(method, url string, c *config.Configuration) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.UserName, c.Password)

	return http.DefaultClient.Do(req)
}

func getFiles(c *config.Configuration) ([]wp, error) {
	fileInfo, err := createURL(c, "/fileinfo")
	if err != nil {
		return []wp{}, fmt.Errorf("can't parse remote: %w", err)
	}

	resp, err := reqWithAuth("GET", fileInfo.String(), c)
	if err != nil {
		return []wp{}, fmt.Errorf("failed to get fileinfo: %w", err)
	}

	defer resp.Body.Close()

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, resp.Body)

	if err != nil {
		return []wp{}, fmt.Errorf("failed to copy: %w", err)
	}

	var files []wp
	err = json.Unmarshal(buf.Bytes(), &files)
	if err != nil {
		return []wp{}, fmt.Errorf("couldn't parse json: %w", err)
	}
	return files, nil
}

func delFile(u fmt.Stringer, c *config.Configuration) error {
	delResp, err := reqWithAuth("DELETE", u.String(), c)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", u.String(), err)
	}
	defer delResp.Body.Close()

	return nil
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

func downloadFile(remote, local string, c *config.Configuration) error {
	dir, fName := filepath.Split(local)
	err := os.MkdirAll(dir, 0775)
	if err != nil {
		return fmt.Errorf("couldn't create dir: %w", err)
	}

	postfix, err := randomString(postfixLen)
	if err != nil {
		return fmt.Errorf("couldn't generate postfix: %w", err)
	}

	tmpFile := path.Join(dir, fmt.Sprintf(".%s.%s", fName, postfix))
	output, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("couldn't create file: %w", err)
	}

	defer func() {
		_ = output.Close()
		_, err := os.Stat(tmpFile)
		if err != nil {
			if os.IsNotExist(err) {
				return
			}
			panic(err)
		}
		os.Remove(tmpFile)
	}()

	resp, err := reqWithAuth("GET", remote, c)
	if err != nil {
		return fmt.Errorf("couldn't download %s: %w", remote, err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("failed downloading %s: %w", remote, err)
	}
	err = output.Close()
	if err != nil {
		return fmt.Errorf("failed to close %s: %w", tmpFile, err)
	}
	err = os.Rename(tmpFile, local)
	if err != nil {
		return fmt.Errorf("couldn't rename %s to %s: %w", tmpFile, local, err)
	}

	return nil
}

func getFile(f wp, c *config.Configuration) error {
	localFile := findLocal(f.WebPath, c)
	if localFile == "" {
		return fmt.Errorf("couldn't find config for remote file: %s", f)
	}

	fileURL, err := createURL(c, f.WebPath)
	if err != nil {
		return fmt.Errorf("couldn't parse remote: %w", err)
	}

	err = downloadFile(fileURL.String(), localFile, c)
	if err != nil {
		return err
	}

	err = delFile(fileURL, c)
	return err
}

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	lock, err := lockfile.New(lockFile)
	if err != nil {
		panic(err)
	}

	if err := lock.TryLock(); err != nil {
		panic(fmt.Sprintf("Can't lock %q, reason %v", lock, err))
	}

	defer func() {
		if err := lock.Unlock(); err != nil {
			logger.Printf("Can't unlock %q, reason %v", lock, err)
		}
	}()

	c, err := config.GetConfig()
	if err != nil {
		logger.Printf("Can't get configuration: %s", err)
		return
	}

	r, err := report.New(c)
	if err != nil {
		logger.Printf("can't send telegram messages: %v", err)
		return
	}
	defer func() {
		err := r.SendReport()
		if err != nil {
			panic(err)
		}
	}()

	files, err := getFiles(c)
	if err != nil {
		e := fmt.Errorf("couldn't get file list: %w", err)
		r.AddError(e)
		logger.Println(e)
		return
	}

	for _, f := range files {
		err := getFile(f, c)
		if err != nil {
			r.AddError(err)
			continue
		}
		r.AddFile(path.Base(f.WebPath))
	}
}
