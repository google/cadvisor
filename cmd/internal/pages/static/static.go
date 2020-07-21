// Copyright 2014 Google Inc. All Rights Reserved.
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

// Handler for /static content.

package static

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"

	"k8s.io/klog/v2"
)

const StaticResource = "/static/"

var popper, _ = Asset("cmd/internal/pages/assets/js/popper.min.js")
var bootstrapJs, _ = Asset("cmd/internal/pages/assets/js/bootstrap-4.0.0-beta.2.min.js")
var containersJs, _ = Asset("cmd/internal/pages/assets/js/containers.js")
var loaderJs, _ = Asset("cmd/internal/pages/assets/js/loader.js")
var jqueryJs, _ = Asset("cmd/internal/pages/assets/js/jquery-3.5.1.min.js")

var bootstrapCss, _ = Asset("cmd/internal/pages/assets/styles/bootstrap-4.0.0-beta.2.min.css")
var bootstrapThemeCss, _ = Asset("cmd/internal/pages/assets/styles/bootstrap-theme-3.1.1.min.css")
var containersCss, _ = Asset("cmd/internal/pages/assets/styles/containers.css")

var staticFiles = map[string][]byte{
	"popper.min.js":                  popper,
	"bootstrap-4.0.0-beta.2.min.css": bootstrapCss,
	"bootstrap-4.0.0-beta.2.min.js":  bootstrapJs,
	"bootstrap-theme-3.1.1.min.css":  bootstrapThemeCss,
	"containers.css":                 containersCss,
	"containers.js":                  containersJs,
	"loader.js":                      loaderJs,
	"jquery-3.5.1.min.js":            jqueryJs,
}

func HandleRequest(w http.ResponseWriter, u *url.URL) {
	if len(u.Path) <= len(StaticResource) {
		http.Error(w, fmt.Sprintf("unknown static resource %q", u.Path), http.StatusNotFound)
		return
	}

	// Get the static content if it exists.
	resource := u.Path[len(StaticResource):]
	content, ok := staticFiles[resource]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown static resource %q", u.Path), http.StatusNotFound)
		return
	}

	// Set Content-Type if we were able to detect it.
	contentType := mime.TypeByExtension(path.Ext(resource))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	if _, err := w.Write(content); err != nil {
		klog.Errorf("Failed to write response: %v", err)
	}
}
