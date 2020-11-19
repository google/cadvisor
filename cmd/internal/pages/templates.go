// Copyright 2020 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// generated by build/assets.sh; DO NOT EDIT

// Code generated by go-bindata. DO NOT EDIT.
// sources:
// cmd/internal/pages/assets/html/containers.html (9.604kB)

package pages

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes  []byte
	info   os.FileInfo
	digest [sha256.Size]byte
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _cmdInternalPagesAssetsHtmlContainersHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xcc\x5a\xcd\x72\xdb\x38\x12\x3e\x4b\x4f\xd1\xc3\xda\xc3\x6c\x55\x48\xd9\x89\xf7\xb0\x59\x59\x55\x1a\x25\xd9\xd1\x8e\x63\xa7\x2c\x7b\xa6\xe6\x08\x92\x2d\x12\x31\x44\x60\x00\x50\xb2\xd6\xe5\x77\xdf\x02\x40\x4a\xfc\x95\xe2\x9f\x4a\x56\x17\x4b\x04\xba\xfb\xeb\xaf\xbb\x81\x06\xe1\xf1\x4f\xbe\x3f\x04\x98\x71\xb1\x95\x34\x49\x35\xbc\x3d\x39\x3d\x83\x7f\x73\x9e\x30\x84\x79\x16\x05\x30\x65\x0c\xae\xcd\x90\x82\x6b\x54\x28\xd7\x18\x07\xc3\x21\xc0\x05\x8d\x30\x53\x18\x43\x9e\xc5\x28\x41\xa7\x08\x53\x41\xa2\x14\xcb\x91\x37\xf0\x3b\x4a\x45\x79\x06\x6f\x83\x13\xf8\xd9\x4c\xf0\x8a\x21\xef\xef\xff\x1a\x02\x6c\x79\x0e\x2b\xb2\x85\x8c\x6b\xc8\x15\x82\x4e\xa9\x82\x25\x65\x08\x78\x1f\xa1\xd0\x40\x33\x88\xf8\x4a\x30\x4a\xb2\x08\x61\x43\x75\x6a\xcd\x14\x4a\x82\x21\xc0\x9f\x85\x0a\x1e\x6a\x42\x33\x20\x10\x71\xb1\x05\xbe\xac\xce\x03\xa2\x0d\x5e\xf3\x49\xb5\x16\xef\x47\xa3\xcd\x66\x13\x10\x8b\x35\xe0\x32\x19\x31\x37\x4f\x8d\x2e\xe6\xb3\x8f\x97\x8b\x8f\xfe\xdb\xe0\xc4\x48\xdc\x66\x0c\x95\x02\x89\x7f\xe5\x54\x62\x0c\xe1\x16\x88\x10\x8c\x46\x24\x64\x08\x8c\x6c\x80\x4b\x20\x89\x44\x8c\x41\x73\x83\x76\x23\xa9\xa6\x59\xf2\x06\x14\x5f\xea\x0d\x91\x38\x04\x88\xa9\xd2\x92\x86\xb9\xae\x51\x55\x62\xa3\xaa\x36\x81\x67\x40\x32\xf0\xa6\x0b\x98\x2f\x3c\xf8\x65\xba\x98\x2f\xde\x0c\x01\xfe\x98\xdf\xfc\x7a\x75\x7b\x03\x7f\x4c\xaf\xaf\xa7\x97\x37\xf3\x8f\x0b\xb8\xba\x86\xd9\xd5\xe5\x87\xf9\xcd\xfc\xea\x72\x01\x57\x9f\x60\x7a\xf9\x27\xfc\x36\xbf\xfc\xf0\x06\x90\xea\x14\x25\xe0\xbd\x90\x06\x3f\x97\x40\x0d\x89\x26\x6e\x00\x0b\xc4\x1a\x80\x25\x77\x80\x94\xc0\x88\x2e\x69\x04\x8c\x64\x49\x4e\x12\x84\x84\xaf\x51\x66\x34\x4b\x40\xa0\x5c\x51\x65\x42\xa9\x80\x64\xf1\x10\x80\xd1\x15\xd5\x44\xdb\x27\x2d\xa7\x82\xa1\xef\x4f\x86\xc3\x71\xaa\x57\x6c\x32\x04\x18\xa7\x48\xe2\x89\x0d\xc1\x58\x53\xcd\x70\x12\x4d\xe3\x35\x55\x5c\x82\x0f\x0f\x0f\xc1\x07\xaa\x04\x23\xdb\x4b\xb2\xc2\xc7\xc7\xf1\xc8\x4d\x71\xd3\x55\x24\xa9\xd0\xa0\x64\x74\xee\x3d\x3c\x04\xd7\x9c\xeb\xc7\x47\x65\x2c\x47\x23\xc1\x85\x40\x19\xac\x68\x16\x7c\x55\xde\x64\x3c\x72\x93\x0b\xc9\x9f\x7c\x1f\x2e\x88\x46\xa5\x6d\x0e\x51\x86\xb1\xc1\x0e\x2b\x9a\xd1\x25\xc5\x18\x66\x8b\x05\x18\x9c\x76\x36\xa3\xd9\x1d\x48\x64\xe7\x9e\xd2\x5b\x86\x2a\x45\xd4\x1e\xa4\x12\x97\x6d\xbb\x21\xe7\x5a\x69\x49\x84\x7f\x16\x9c\x04\x27\x7e\x88\x9a\x04\x6f\x2d\x8e\x48\x29\x6f\x32\xdc\x03\xb8\x12\x86\x22\xc2\x0c\x3b\x2b\x7c\xa9\x39\xab\xc4\x7f\x17\x9c\x06\xa7\x2d\x6b\x4f\xd1\x18\xf1\xcc\x54\x0b\x4a\xd5\x02\x7c\x90\xb1\xff\x90\x35\x59\xb8\x80\xec\x3c\x39\x14\xa0\xaf\x7f\xe5\x28\xb7\xfe\xbb\xe0\x1f\x05\xe0\x8e\x30\x1d\x92\x3f\x40\x74\xbf\x26\xbd\x15\x78\xee\x69\xbc\xd7\xa3\xaf\x64\x4d\xdc\x53\xaf\xdb\x40\x62\x97\x39\xff\xab\x22\x82\x36\x54\x3e\x5b\x67\x85\xdc\x57\x02\x19\xa5\x44\xea\xb6\xb6\xf1\xa8\x2c\xab\x71\xc8\xe3\x6d\x61\x20\xa6\x6b\x88\x18\x51\xea\xdc\xdb\x21\x71\xd9\xe7\xab\x94\x6f\x22\xa2\xd0\x83\x49\xb1\x1c\x8e\x49\x33\x43\xbc\xbd\x30\xf3\xd5\xca\x3f\x7d\xeb\x01\x8d\xcf\x3d\xc6\x13\xee\xed\xc4\x46\x64\xf7\xb5\x66\xaf\x14\x99\x0c\x07\xd5\x01\x41\x12\xf4\x0d\x58\x94\x66\xc8\x2c\x08\xa7\x93\x76\xdd\xa7\xa7\x46\x6e\x14\xd3\xb5\xf9\xcb\x59\x29\x1e\x4a\x24\x71\x24\xf3\x55\xe8\xa4\x1f\x1e\x24\xc9\x12\x84\xbf\x09\x22\x31\xd3\xb3\x9d\x9b\xef\xcf\x21\xf8\x52\x7f\xa6\x1e\x1f\xad\x41\x46\x27\x15\x67\x9b\x92\xc1\x05\xcd\xee\x1e\x1f\xbd\x49\xc7\xd0\x0d\xde\x6b\x83\x8e\x4c\xc6\x23\x46\x0b\x00\x98\xc5\x46\xf1\x78\xc4\xd9\x9e\x14\x0b\xdc\xfd\x78\x78\xa0\x4b\x08\xe6\xca\x91\x7a\x84\x2b\x28\x3e\xe3\xf4\x6c\x0f\x32\x08\x46\x31\x8f\xee\x0c\x63\x1f\xec\x5f\xd8\xfb\xe4\xc0\xa4\x67\x3d\xa6\x1d\xb8\x2a\x90\x45\x1e\x46\x55\x46\x5e\x16\xbb\x77\x93\x9a\xbe\xf1\x28\x7d\x57\x0d\x5c\x45\x98\x51\xa5\xfd\x44\xf2\x5c\x34\x22\xa7\x2a\x0a\x6c\xd8\x9a\x08\x07\xb5\xe4\xac\xcd\x2f\x83\xd5\x36\xe2\x53\x8d\x2b\x1b\xc4\xda\xfc\x7d\x04\x1b\xc1\xab\xb0\xd6\x4f\xa1\x63\xd0\xc5\x60\xa1\x89\xce\x5f\x83\xc0\x0f\x92\xae\x51\x82\xd3\xd7\x24\x30\x67\x47\xf9\x73\xa9\xa1\xac\xb8\xe5\xaf\x81\xcf\xa5\xbc\x53\x03\x1d\x14\x8d\x95\x20\x59\x69\xc5\xa8\xf1\x19\x09\x91\x59\xee\xaa\xba\x83\xdf\x70\x6b\xa8\x33\xd3\x27\xd0\x1c\xfc\x9d\xb0\xdc\x56\x6e\xb3\x2e\xea\xac\x39\x67\xf7\xd8\x06\xcf\x83\xb6\xd0\x5c\x92\x04\xc7\xa1\x9c\x14\x80\x8c\xaa\x3e\xb2\x06\x7b\xae\xac\xf9\x16\x57\xfd\xa8\x9e\xca\x57\x45\x7f\x9b\xaf\xea\x60\x9d\xaf\xc1\x8e\xae\xc1\x78\x94\x33\xeb\x4d\xc9\x64\xf1\xa0\x2f\x5b\xbb\x6a\xdc\x79\x35\x5f\x91\x04\x8f\x67\x28\xec\x3e\xfd\xa9\x0a\x95\x8f\xc9\x59\xa7\xda\x25\x6b\x65\xa4\x8a\xcb\x69\x33\xfb\x85\xcb\x13\x9f\x5a\x19\xb3\x6f\xd5\x66\x99\x10\x86\x72\xff\xfb\x98\x6f\xd7\xa8\x78\x2e\x23\x54\xd3\x35\xa1\xcc\x74\xdf\xaf\x50\x83\x73\xc5\x99\xed\x60\x1b\xf5\xe7\x4c\xce\x44\x5e\x35\xd6\x9b\x68\x15\x26\x7a\xf3\x07\x48\xa4\xe9\xda\xf4\xfa\x85\x45\xdf\xb6\xb8\x20\x48\x86\xcc\x7d\xf7\x26\xb3\x2f\xb7\x2e\xfc\x7b\x8d\xc5\xe2\x2d\x30\x32\x70\x82\x0b\xd3\x73\xef\x1c\x3f\x6c\xf2\x50\x1d\xa5\x44\x9a\x38\x96\x39\x2a\x24\xcd\xb4\x7b\xd8\x36\x06\x35\x35\x79\x46\x77\x6a\x54\x55\x4d\x1b\x79\x35\x88\x1d\xbe\x7c\x26\xf7\xaf\xe4\xce\x67\x72\x0f\x56\x55\xc3\xa3\x19\xaf\x3b\xb4\xb7\xd8\xef\x53\xc4\x5f\xe4\x92\xba\x7b\xb9\x3b\x53\xc6\xf8\xc6\x9c\x4e\x78\x3b\x48\xc6\x42\xc3\x20\x04\x9f\x49\x94\xd2\x0c\xe7\xd9\x92\x07\x97\xf9\xca\xca\x95\x6b\x4c\x1b\x7d\xb9\xd4\xec\x7e\x3b\x27\x3e\xe3\x8a\xcb\xed\xf7\x4d\x78\x67\xf3\x40\xce\xbb\x09\x81\x7b\xe9\x60\xd5\xbc\x9c\xde\x8a\xb2\x66\x05\xd0\xff\xe2\x01\xc3\xfd\x49\x53\xc8\xdf\x66\x54\x1f\x90\x7f\x4e\x56\x15\x7a\x5e\xa9\x50\xba\x8a\xa4\xed\xf4\xd1\x1a\xe9\x75\xb7\x90\x7c\x81\xa3\x8b\x0d\x11\xaf\xb5\xc8\x6d\x88\xe8\x5c\x16\xda\x1e\x57\xac\x3e\xc3\xeb\x8a\xf4\x11\xcf\x9b\xa5\x57\x78\x57\xeb\x42\x9f\xbd\x99\xdd\x2a\xd3\x1a\xf5\x77\xe2\xb6\xf2\x8a\xfa\x13\x92\xae\x88\xdc\x1e\x68\x03\xcc\x2c\x63\x81\x66\x49\xbb\x11\xa8\x4f\x2b\x8a\xf9\x6a\x8d\x72\x4d\x71\x73\xb8\x3d\xa8\x76\x08\xb9\x41\xec\x27\x24\x4f\xd0\xab\xab\x34\xa7\xd9\x5d\xcb\xf0\x43\xbc\xf9\x22\x79\x84\x4a\x1d\xeb\x76\xaa\xee\x88\x52\xc4\xd7\x5c\x7c\x93\x43\x3d\x7d\xc6\x77\x74\xd3\xb6\x1c\xdf\xe2\x60\x87\x37\x0d\x03\x67\x93\x1b\xae\x09\x83\x32\x0f\xcf\x6c\x66\x56\xf8\x89\x44\xee\x6b\x33\xc5\x77\x81\xb7\x2f\x35\xf6\xa4\x40\xf9\x02\xca\xa8\x9a\x7d\xb9\x85\x0b\x4e\x62\x98\xae\x51\x1e\xd0\xc7\x38\x89\xeb\x8a\x76\xef\xa5\xaa\xc8\x2c\x26\x10\xf6\x08\x2d\x7b\x95\x09\x94\xbe\xd9\xff\x3b\xf1\x75\xab\xfc\x45\x22\xb9\x8b\xf9\x26\xeb\xd3\xe9\x54\x85\xe5\xb4\x5e\xa5\xed\xd4\x38\xba\x3b\x7f\xc7\x34\x29\x37\xea\xef\x94\x29\x2b\x6b\xee\x78\x18\x42\x39\x6a\x3c\xa9\x00\x90\x7c\x03\xdd\x07\x9e\x83\x21\x6c\x4c\x6b\x2f\xc7\xff\xb4\x67\xcb\x9a\xab\x92\x27\x12\xed\x6b\x54\x68\x7d\xba\x26\xfa\x21\x91\x50\xfd\xe1\xc7\xe6\xa0\x2a\xbd\x72\x1d\x71\x03\x29\xd7\xbe\xa3\xa2\x53\x33\xd4\xf7\x2a\x25\x7d\x9e\xb1\xad\x37\xf9\x95\x6b\x28\x03\xe6\x0e\xc9\x1d\x92\x6d\x36\x9f\x02\x97\x66\x4b\xde\x00\x1b\x71\x16\x3f\x07\xed\x8c\xb3\xf8\x5b\xe1\x0e\x06\x9d\xb8\xbb\x1f\xb6\x23\xf7\xce\xab\x66\x97\xc6\xfb\xe6\xea\xf3\xc4\xa2\xbc\x44\xbd\xe1\xf2\xee\x89\x55\x39\x78\x79\x39\x16\x86\x8b\xcd\xfe\x29\x85\x38\x68\x8e\xc6\x92\x0b\x93\xfc\xed\x02\x09\x73\xad\xf9\x2e\x5e\xa1\xce\x20\xd4\x99\x1f\xe3\x92\xe4\x4c\x43\x29\xe7\x6b\x9e\x24\x0c\xbd\xe2\x7d\xb6\x13\x72\x3c\x67\x0e\xa5\xaf\x90\x61\x64\x8f\x00\x3b\x63\x10\x13\x4d\x0a\xd1\x0a\x06\x20\x92\x12\x3f\x25\x4a\x70\x91\x8b\x73\x4f\xcb\x1c\x8b\x87\x78\x2f\x48\x16\x63\x7c\xee\x2d\x09\x53\xd8\x91\x62\x2e\xbd\xba\x0d\x97\xb1\xee\xce\xaf\x5a\x62\x46\x44\x62\x65\xee\xa0\xcc\x04\xe7\x59\x8b\xa5\x9c\x75\x9b\xf4\x9a\x04\xfb\x2b\xcc\x72\x0f\x24\x37\x1e\xbb\xef\xd6\x31\xdb\x5d\x32\x8c\xc3\xed\x41\xc6\xda\x39\x5f\xbc\x1e\x3a\x90\xb6\x4f\x59\x90\x53\xc9\xf3\x24\x15\xb9\x6e\xaf\x82\xbb\x65\xb9\x84\x17\x6e\x35\xaa\xf6\xf6\xfd\x0c\xb3\x1f\xa5\xe4\xf6\xf5\x71\x6b\x0b\x28\x6d\xa1\x9d\xd1\x6f\xac\xe1\x7c\xa3\x42\x3f\xa9\x1f\xb6\x65\x7e\xa2\x0c\xd5\x56\x69\x5c\x7d\x7b\x07\xb9\xdc\xc9\xb8\xbd\xaf\xb3\x89\xec\xd7\xd4\xb3\x4c\xcd\x72\xa5\xf9\xea\x33\x6a\x49\xa3\xa7\xf2\x71\x64\xb1\x1a\x1c\x62\x60\xea\x2e\xca\x4d\x1e\x43\x61\xbd\xb9\x62\x1d\xca\x95\x46\x2f\x65\x9d\xf0\x57\x4e\xcf\xd1\x7c\x18\x34\x0f\x9b\x1d\xb7\x20\x3f\x2c\x35\x3a\xee\x4e\x8e\x65\xc7\xb7\x35\x55\x02\x4c\xdf\x6c\xdb\x9a\xf7\xcd\xf5\x82\x66\x22\xd7\xb5\x56\xb7\x7a\x43\xe2\xc7\xee\x22\xce\x8f\x78\x9e\x69\xaf\x73\xff\xde\x6d\xdd\x5d\x72\x56\x7d\x8f\xdc\x9a\xb0\x1c\xcf\x4f\x4f\x1a\x90\xfb\x17\x9a\x4e\x84\xb5\x6e\xb0\xa1\xa9\x7b\x01\x7c\x26\x87\xae\x19\x39\x4a\x63\xd1\x46\xfc\x7f\x32\x59\x6b\xb5\x9c\x15\xc9\x19\xab\x98\x09\x19\x8f\xee\x9a\x0c\xb4\xf7\xc7\x66\x4f\xfe\x8a\x61\xe9\x59\xba\x3b\x06\xab\x43\x95\x81\xc3\x57\xe9\xa5\xb0\xd2\x44\xea\x2f\x24\xc1\x9f\x1f\x1e\x82\xdd\x0d\xaa\xbb\x71\x7e\x03\xe6\x59\xed\xfc\x6d\x1f\xb5\x8e\x5b\xf6\xa9\xbb\xca\xb5\x5f\xcb\x7b\x5d\xfb\x4f\x4c\xe6\x13\x4b\xb2\x71\xd7\x23\xc6\x4c\xfd\x26\xa6\x98\x54\xbf\xb9\x77\x17\xf6\xe3\x91\xfb\x0f\x99\xff\x05\x00\x00\xff\xff\x77\x17\x4e\xd5\x84\x25\x00\x00")

func cmdInternalPagesAssetsHtmlContainersHtmlBytes() ([]byte, error) {
	return bindataRead(
		_cmdInternalPagesAssetsHtmlContainersHtml,
		"cmd/internal/pages/assets/html/containers.html",
	)
}

func cmdInternalPagesAssetsHtmlContainersHtml() (*asset, error) {
	bytes, err := cmdInternalPagesAssetsHtmlContainersHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "cmd/internal/pages/assets/html/containers.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x45, 0xa4, 0x2, 0xb9, 0xc6, 0x15, 0xae, 0xba, 0xe4, 0x35, 0x28, 0xc2, 0x30, 0x69, 0xa2, 0x85, 0x30, 0x2d, 0x70, 0xde, 0x7, 0x98, 0x4a, 0x32, 0xfa, 0xe6, 0x60, 0xc0, 0xf5, 0x70, 0xdc, 0x97}}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetString returns the asset contents as a string (instead of a []byte).
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// MustAssetString is like AssetString but panics when Asset would return an
// error. It simplifies safe initialization of global variables.
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetDigest returns the digest of the file with the given name. It returns an
// error if the asset could not be found or the digest could not be loaded.
func AssetDigest(name string) ([sha256.Size]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s can't read by error: %v", name, err)
		}
		return a.digest, nil
	}
	return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s not found", name)
}

// Digests returns a map of all known files and their checksums.
func Digests() (map[string][sha256.Size]byte, error) {
	mp := make(map[string][sha256.Size]byte, len(_bindata))
	for name := range _bindata {
		a, err := _bindata[name]()
		if err != nil {
			return nil, err
		}
		mp[name] = a.digest
	}
	return mp, nil
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"cmd/internal/pages/assets/html/containers.html": cmdInternalPagesAssetsHtmlContainersHtml,
}

// AssetDebug is true if the assets were built with the debug flag enabled.
const AssetDebug = false

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"},
// AssetDir("data/img") would return []string{"a.png", "b.png"},
// AssetDir("foo.txt") and AssetDir("notexist") would return an error, and
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"cmd": {nil, map[string]*bintree{
		"internal": {nil, map[string]*bintree{
			"pages": {nil, map[string]*bintree{
				"assets": {nil, map[string]*bintree{
					"html": {nil, map[string]*bintree{
						"containers.html": {cmdInternalPagesAssetsHtmlContainersHtml, map[string]*bintree{}},
					}},
				}},
			}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory.
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively.
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}