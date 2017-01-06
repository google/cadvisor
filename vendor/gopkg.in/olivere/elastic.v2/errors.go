// Copyright 2012-2015 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	// ErrMissingIndex is returned e.g. from DeleteService if the index is missing.
	ErrMissingIndex = errors.New("elastic: index is missing")

	// ErrMissingType is returned e.g. from DeleteService if the type is missing.
	ErrMissingType = errors.New("elastic: type is missing")

	// ErrMissingId is returned e.g. from DeleteService if the document identifier is missing.
	ErrMissingId = errors.New("elastic: id is missing")
)

func checkResponse(res *http.Response) error {
	// 200-299 and 404 are valid status codes
	if (res.StatusCode >= 200 && res.StatusCode <= 299) || res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.Body == nil {
		return fmt.Errorf("elastic: Error %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
	}
	slurp, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("elastic: Error %d (%s) when reading body: %v", res.StatusCode, http.StatusText(res.StatusCode), err)
	}
	errReply := new(Error)
	err = json.Unmarshal(slurp, errReply)
	if err != nil {
		return fmt.Errorf("elastic: Error %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
	}
	if errReply != nil {
		if errReply.Status == 0 {
			errReply.Status = res.StatusCode
		}
		return errReply
	}
	return fmt.Errorf("elastic: Error %d (%s)", res.StatusCode, http.StatusText(res.StatusCode))
}

type Error struct {
	Status  int    `json:"status"`
	Message string `json:"error"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("elastic: Error %d (%s): %s", e.Status, http.StatusText(e.Status), e.Message)
	} else {
		return fmt.Sprintf("elastic: Error %d (%s)", e.Status, http.StatusText(e.Status))
	}
}
