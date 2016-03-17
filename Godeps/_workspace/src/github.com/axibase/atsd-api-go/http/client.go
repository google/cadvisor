/*
* Copyright 2015 Axibase Corporation or its affiliates. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License").
* You may not use this file except in compliance with the License.
* A copy of the License is located at
*
* https://www.axibase.com/atsd/axibase-apache-2.0.pdf
*
* or in the "license" file accompanying this file. This file is distributed
* on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
* express or implied. See the License for the specific language governing
* permissions and limitations under the License.
 */

package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"net/url"
	"strconv"
)

const (
	seriesQueryPath  = "/api/v1/series"
	seriesInsertPath = "/api/v1/series/insert"

	messagesQueryPath  = "/api/v1/messages"
	messagesInsertPath = "/api/v1/messages/insert"

	propertiesInsertPath = "/api/v1/properties/insert"

	entitiesPath      = "/api/v1/entities"
	entitiesGroupPath = "/api/v1/entity-groups"

	metricsPath = "/api/v1/metrics"
	commandPath = "/api/v1/command"

	sql = "/sql"
)

type Client struct {
	url      *url.URL
	username string
	password string

	Series       *seriesApi
	Properties   *propertiesApi
	Entities     *entitiesApi
	EntityGroups *entityGroupsApi
	Messages     *messagesApi

	Metric *metricApi

	SQL *sqlApi

	httpClient *http.Client
}

func New(mUrl url.URL, username, password string) *Client {
	var client = Client{url: &mUrl, username: username, password: password}
	client.Series = &seriesApi{&client}
	client.Properties = &propertiesApi{&client}
	client.Entities = &entitiesApi{&client}
	client.EntityGroups = &entityGroupsApi{&client}
	client.Messages = &messagesApi{&client}
	client.Metric = &metricApi{&client}
	client.SQL = &sqlApi{&client}
	client.httpClient = &http.Client{}
	return &client
}

func (self *Client) Url() url.URL {
	return *self.url
}
func (self *Client) request(reqType, apiUrl string, reqJson []byte) (string, error) {
	req, err := http.NewRequest(reqType, self.url.String(), bytes.NewReader(reqJson))
	req.URL.Opaque = req.URL.Path + apiUrl //todo: check
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(self.username, self.password)
	res, err := self.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	jsonData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var error struct {
		Error string `json:"error"`
	}

	_ = json.Unmarshal(jsonData, &error)

	if error.Error != "" {
		return string(jsonData), errors.New(error.Error)
	}

	return string(jsonData), nil
}

type seriesApi struct {
	client *Client
}

func (self *seriesApi) Query(queries []*SeriesQuery) ([]*Series, error) {

	request := struct {
		Queries []*SeriesQuery `json:"queries"`
	}{queries}

	jsonRequest, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	jsonData, err := self.client.request("POST", seriesQueryPath, jsonRequest)
	if err != nil {
		return nil, err
	}
	var series struct {
		Series []*Series `json:"series"`
	}
	err = json.Unmarshal([]byte(jsonData), &series)
	if err != nil {
		panic(err)
	}
	for _, s := range series.Series {
		if s.Warning != "" {
			glog.Warning(s.Warning)
		}
	}
	return series.Series, nil
}

func (self *seriesApi) Insert(series []*Series) error {
	jsonSeries, err := json.Marshal(series)
	if err != nil {
		panic(err)
	}
	_, err = self.client.request("POST", seriesInsertPath, jsonSeries)
	if err != nil {
		return err
	}

	return nil
}

type propertiesApi struct {
	client *Client
}

func (self *propertiesApi) Insert(properties []*Property) error {
	jsonProperties, err := json.Marshal(properties)
	if err != nil {
		panic(err)
	}
	_, err = self.client.request("POST", propertiesInsertPath, jsonProperties)
	if err != nil {
		return err
	}

	return nil
}

type entitiesApi struct {
	client *Client
}

func (self *entitiesApi) Create(entity *Entity) error {
	jsonRequest, err := json.Marshal(entity)
	if err != nil {
		panic(err)
	}
	mUrl := url.URL{}
	mUrl.Path = entitiesPath + "/" + entity.Name()
	_, err = self.client.request("PUT", mUrl.String(), jsonRequest)
	if err != nil {
		return err
	}
	return nil
}

func (self *entitiesApi) Update(entity *Entity) error {
	jsonRequest, err := json.Marshal(entity)
	if err != nil {
		panic(err)
	}
	mUrl := url.URL{}
	mUrl.Path = entitiesPath + "/" + entity.Name()
	_, err = self.client.request("PATCH", mUrl.String(), jsonRequest)
	if err != nil {
		return err
	}
	return nil
}
func (self *entitiesApi) List(expression string, tags []string, limit uint64) ([]*Entity, error) {
	tagsParams := ""
	if len(tags) == 1 && tags[0] == "*" {
		tagsParams = "*"
	} else {
		for i, tag := range tags {
			if i == 0 {
				tagsParams += tag
			} else {
				tagsParams += "," + tag
			}
		}
	}

	mUrl := url.URL{}
	mUrl.Path = entitiesPath
	q := url.Values{}
	q.Set("tags", tagsParams)
	q.Set("expression", expression)
	q.Set("limit", strconv.FormatUint(limit, 10))
	mUrl.RawQuery = q.Encode()
	jsonData, err := self.client.request("GET", mUrl.String(), []byte{})
	if err != nil {
		return nil, err
	}

	var entities []*Entity
	err = json.Unmarshal([]byte(jsonData), &entities)
	if err != nil {
		panic(err)
	}

	return entities, nil
}

type metricApi struct {
	client *Client
}

func (self *metricApi) CreateOrReplace(metric *Metric) error {
	jsonRequest, err := json.Marshal(metric)
	if err != nil {
		panic(err)
	}
	mUrl := url.URL{}
	mUrl.Path = metricsPath + "/" + metric.Name()
	_, err = self.client.request("PUT", mUrl.String(), jsonRequest)
	if err != nil {
		return err
	}
	return nil
}

type messagesApi struct {
	client *Client
}

func (self *messagesApi) Insert(messages []*Message) error {
	jsonRequest, err := json.Marshal(messages)
	if err != nil {
		panic(err)
	}
	_, err = self.client.request("POST", messagesInsertPath, jsonRequest)
	if err != nil {
		return err
	}
	return nil
}
func (self *messagesApi) Query(query *MessagesQuery) ([]*Message, error) {
	jsonRequest, err := json.Marshal(query)
	if err != nil {
		panic(err)
	}
	jsonData, err := self.client.request("POST", messagesQueryPath, jsonRequest)
	if err != nil {
		return nil, err
	}
	var messages []*Message
	err = json.Unmarshal([]byte(jsonData), &messages)
	if err != nil {
		panic(err)
	}
	return messages, nil
}

type entityGroupsApi struct {
	client *Client
}

func (self *entityGroupsApi) EntitiesList(group, expression string, tags []string, limit uint64) ([]*Entity, error) {
	tagsParams := ""
	if len(tags) == 1 && tags[0] == "*" {
		tagsParams = "*"
	} else {
		for i, tag := range tags {
			if i == 0 {
				tagsParams += tag
			} else {
				tagsParams += "," + tag
			}
		}
	}
	mUrl := url.URL{}
	mUrl.Path = entitiesGroupPath + "/" + group + "/entities"
	q := url.Values{}
	q.Add("tags", tagsParams)
	q.Add("expression", expression)
	q.Add("limit", strconv.FormatUint(limit, 10))
	mUrl.RawQuery = q.Encode()
	jsonData, err := self.client.request("GET", mUrl.String(), []byte{})
	if err != nil {
		return nil, err
	}

	var entities []*Entity
	err = json.Unmarshal([]byte(jsonData), &entities)
	if err != nil {
		panic(err)
	}

	return entities, nil
}
func (self *entityGroupsApi) List(expression string, tags []string, limit uint64) ([]*EntityGroup, error) {
	tagsParams := ""
	if len(tags) == 1 && tags[0] == "*" {
		tagsParams = "*"
	} else {
		for i, tag := range tags {
			if i == 0 {
				tagsParams += tag
			} else {
				tagsParams += "," + tag
			}
		}
	}

	mUrl := url.URL{}
	mUrl.Path = entitiesGroupPath
	q := url.Values{}
	q.Set("tags", tagsParams)
	q.Set("expression", expression)
	q.Set("limit", strconv.FormatUint(limit, 10))
	mUrl.RawQuery = q.Encode()
	jsonData, err := self.client.request("GET", mUrl.String(), []byte{})
	if err != nil {
		return nil, err
	}

	var entityGroups []*EntityGroup
	err = json.Unmarshal([]byte(jsonData), &entityGroups)
	if err != nil {
		panic(err)
	}

	return entityGroups, nil
}

type sqlApi struct {
	client *Client
}

func (self *sqlApi) Query(query string) (*Table, error) {
	mUrl := url.URL{}
	mUrl.Path = sql

	params := url.Values{}
	params.Set("q", query)
	mUrl.RawQuery = params.Encode()
	jsonData, err := self.client.request("GET", mUrl.String(), []byte{})
	if err != nil {
		return nil, err
	}
	var table *Table
	dec := json.NewDecoder(bytes.NewBufferString(jsonData))
	dec.UseNumber()
	err = dec.Decode(&table)
	if err != nil {
		panic(err)
	}

	return table, nil

}
