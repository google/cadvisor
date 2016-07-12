// Copyright 2016 Google Inc. All Rights Reserved.
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

package amazonsqs

import (
	"flag"
	"os"
	"sync"
	"time"

	"encoding/json"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/utils/container"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

func init() {
	storage.RegisterStorageDriver("amazonsqs", new)
}

var (
	argSqsRegion = flag.String("storage_driver_amazonsqs_region", "", "AWS SQS Region")
	argSqsQueue  = flag.String("storage_driver_amazonsqs_queue", "", "AWS SQS Queue")
)

type sqsStorage struct {
	sqsService  sqsiface.SQSAPI
	machineName string
	lock        sync.Mutex
}

type detailSpec struct {
	Timestamp       time.Time            `json:"timestamp"`
	MachineName     string               `json:"machine_name,omitempty"`
	ContainerName   string               `json:"container_name,omitempty"`
	ContainerID     string               `json:"container_id,omitempty"`
	ContainerLabels map[string]string    `json:"container_labels,omitempty"`
	ContainerStats  *info.ContainerStats `json:"container_stats,omitempty"`
}

func (driver *sqsStorage) infoToDetailSpec(ref info.ContainerReference, stats *info.ContainerStats) *detailSpec {
	timestamp := time.Now()
	containerID := ref.Id
	containerLabels := ref.Labels
	containerName := container.GetPreferredName(ref)

	detail := &detailSpec{
		Timestamp:       timestamp,
		MachineName:     driver.machineName,
		ContainerName:   containerName,
		ContainerID:     containerID,
		ContainerLabels: containerLabels,
		ContainerStats:  stats,
	}
	return detail
}

func (driver *sqsStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	func() {
		// AddStats will be invoked simultaneously from multiple threads
		// and only one of them will perform a write.
		driver.lock.Lock()
		defer driver.lock.Unlock()

		detail := driver.infoToDetailSpec(ref, stats)
		body, err := json.Marshal(detail)

		params := &sqs.SendMessageInput{
			MessageBody: aws.String(string(body)),
			QueueUrl:    argSqsQueue,
		}

		_, err = driver.sqsService.SendMessage(params)

		if err != nil {
			return
		}

	}()

	return nil
}

func (driver *sqsStorage) Close() error {
	//driver.sqsService = nil
	return nil
}

func new() (storage.StorageDriver, error) {
	machineName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(machineName)
}

func newStorage(machineName string) (storage.StorageDriver, error) {

	sess := session.New(&aws.Config{Region: argSqsRegion})

	ret := &sqsStorage{
		machineName: machineName,
		sqsService:  sqs.New(sess),
	}
	return ret, nil
}
