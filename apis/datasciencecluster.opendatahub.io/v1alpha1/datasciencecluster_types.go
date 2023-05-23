/*
Copyright 2022.

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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataScienceClusterSpec defines the desired state of DataScienceCluster
type DataScienceClusterSpec struct {
	// A profile sets the default components and configuration to install for a given
	// use case. The profile configuration can still be overriden by the user on a per
	// component basis. If not defined, the 'full' profile is used. Valid values are:
	// - full: all components are installed
	// - serving: only serving components are installed
	// - training: only training components are installed
	// - workbench: only workbench components are installed
	Profile string `json:"profile,omitempty"`

	// Components is used to override and fine tune specific component configurations.
	Components Components `json:"components,omitempty"`
}

type Components struct {
	// Dashboard component configuration
	Dashboard Dashboard `json:"dashboard,omitempty"`

	// DataServicePipeline component configuration
	Pipeline Pipeline `json:"dataServicePipeline,omitempty"`
}

type Component struct {
	// enables or disables the component. A disabled component will not be installed.
	Enabled bool `json:"enabled,omitempty"`
	// number of replicas to deploy for the component
	Replicas int64 `json:"replicas,omitempty"`
	// resources to allocate to the component
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

type Dashboard struct {
	Component `json:""`
}

type Pipeline struct {
	Component `json:""`
}

// DataScienceClusterStatus defines the observed state of DataScienceCluster
type DataScienceClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DataScienceCluster is the Schema for the datascienceclusters API
type DataScienceCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataScienceClusterSpec   `json:"spec,omitempty"`
	Status DataScienceClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataScienceClusterList contains a list of DataScienceCluster
type DataScienceClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataScienceCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataScienceCluster{}, &DataScienceClusterList{})
}
