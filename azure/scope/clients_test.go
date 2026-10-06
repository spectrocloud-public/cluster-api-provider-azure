/*
Copyright 2026 The Kubernetes Authors.

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

package scope

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/cluster-api-provider-azure/azure"
)

// testCustomCloudEnvJSON mirrors the azure-capz-env.json payload of the azure-capz-env-config ConfigMap.
const testCustomCloudEnvJSON = `{
	"name": "AzureUSSecretCloud",
	"resourceManagerEndpoint": "https://management.azure.example.scloud/",
	"activeDirectoryEndpoint": "https://login.microsoftonline.example.scloud/",
	"tokenAudience": "https://management.azure.example.scloud/",
	"resourceManagerVMDNSSuffix": "cloudapp.example.scloud",
	"graphEndpoint": "https://graph.example.scloud/",
	"galleryEndpoint": "https://gallery.example.scloud/",
	"storageEndpointSuffix": "core.example.scloud"
}`

func TestGetSettingsFromEnvironment(t *testing.T) {
	g := NewWithT(t)

	// The environment registry is process-global and has no unregister, so check the
	// not-yet-registered case before registering.
	err := (&AzureClients{}).getSettingsFromEnvironment(azure.AzSecretCloudName)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring(azure.AzSecretCloudName))
	g.Expect(err.Error()).To(ContainSubstring(AzureEnvConfigMapName))

	// Register the custom cloud the same way InitializeAzureConfigForCluster does.
	g.Expect(processAzureEnvironmentJSON(testCustomCloudEnvJSON)).To(Succeed())

	tests := []struct {
		name                   string
		environmentName        string
		wantCloudEnvironment   string
		wantResourceManager    string
		wantActiveDirectory    string
		wantTokenAudience      string
		wantVMDNSSuffix        string
		wantErr                bool
		wantErrContainsEnvName bool
	}{
		{
			name:                 "empty name defaults to public cloud",
			environmentName:      "",
			wantCloudEnvironment: azure.PublicCloudName,
			wantResourceManager:  cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint,
			wantActiveDirectory:  cloud.AzurePublic.ActiveDirectoryAuthorityHost,
			wantTokenAudience:    cloud.AzurePublic.Services[cloud.ResourceManager].Audience,
			wantVMDNSSuffix:      "cloudapp.azure.com",
		},
		{
			name:                 "US government cloud",
			environmentName:      azure.USGovernmentCloudName,
			wantCloudEnvironment: azure.USGovernmentCloudName,
			wantResourceManager:  cloud.AzureGovernment.Services[cloud.ResourceManager].Endpoint,
			wantActiveDirectory:  cloud.AzureGovernment.ActiveDirectoryAuthorityHost,
			wantTokenAudience:    cloud.AzureGovernment.Services[cloud.ResourceManager].Audience,
			wantVMDNSSuffix:      "cloudapp.usgovcloudapi.net",
		},
		{
			name:                 "US Secret cloud registered from the ConfigMap",
			environmentName:      azure.AzSecretCloudName,
			wantCloudEnvironment: azure.AzSecretCloudName,
			wantResourceManager:  "https://management.azure.example.scloud/",
			wantActiveDirectory:  "https://login.microsoftonline.example.scloud/",
			wantTokenAudience:    "https://management.azure.example.scloud/",
			wantVMDNSSuffix:      "cloudapp.example.scloud",
		},
		{
			name:                   "unregistered cloud name",
			environmentName:        "AzureUnrecognizedCloud",
			wantErr:                true,
			wantErrContainsEnvName: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			c := &AzureClients{}

			err := c.getSettingsFromEnvironment(tc.environmentName)
			if tc.wantErr {
				g.Expect(err).To(HaveOccurred())
				if tc.wantErrContainsEnvName {
					g.Expect(err.Error()).To(ContainSubstring(tc.environmentName))
				}
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(c.CloudEnvironment()).To(Equal(tc.wantCloudEnvironment))
			g.Expect(c.ResourceManagerEndpoint).To(Equal(tc.wantResourceManager))
			g.Expect(c.activeDirectoryEndpoint).To(Equal(tc.wantActiveDirectory))
			g.Expect(c.tokenAudience).To(Equal(tc.wantTokenAudience))
			g.Expect(c.ResourceManagerVMDNSSuffix).To(Equal(tc.wantVMDNSSuffix))
		})
	}
}
