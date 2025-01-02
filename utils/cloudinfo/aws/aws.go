// Copyright 2015 Google Inc. All Rights Reserved.
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

package cloudinfo

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/cloudinfo"
)

const (
	productVerFileName       = "/sys/class/dmi/id/product_version"
	biosVerFileName          = "/sys/class/dmi/id/bios_vendor"
	systemdOSReleaseFileName = "/etc/os-release"
	amazon                   = "amazon"
)

func init() {
	cloudinfo.RegisterCloudProvider(info.AWS, &provider{})
}

type provider struct{}

var _ cloudinfo.CloudProvider = provider{}

func (provider) IsActiveProvider() bool {
	return fileContainsAmazonIdentifier(productVerFileName) ||
		fileContainsAmazonIdentifier(biosVerFileName) ||
		fileContainsAmazonIdentifier(systemdOSReleaseFileName)
}

func fileContainsAmazonIdentifier(filename string) bool {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	return strings.Contains(string(fileContent), amazon)
}

func getAwsMetadata(name string) string {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return info.UnknownInstance
	}

	client := imds.NewFromConfig(cfg)
	data, err := client.GetMetadata(context.TODO(), &imds.GetMetadataInput{
		Path: name,
	})
	if err != nil {
		return info.UnknownInstance
	}

	raw, err := io.ReadAll(data.Content)
	if err != nil {
		return info.UnknownInstance
	}

	return string(raw)
}

func (provider) GetInstanceType() info.InstanceType {
	return info.InstanceType(getAwsMetadata("instance-type"))
}

func (provider) GetInstanceID() info.InstanceID {
	return info.InstanceID(getAwsMetadata("instance-id"))
}
