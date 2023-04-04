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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ArgoHelmRepoSecretSpec defines the desired state of ArgoHelmRepoSecret
type ArgoHelmRepoSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ArgoHelmRepoSecret. Edit argohelmreposecret_types.go to remove/update
	GenerateSecretName string           `json:"generated_secret_name"`
	URL                string           `json:"url"`
	Region             string           `json:"region"`
	Frequency          *metav1.Duration `json:"frequency"`
}

// ArgoHelmRepoSecretStatus defines the observed state of ArgoHelmRepoSecret
type ArgoHelmRepoSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase           string       `json:"phase,omitempty"`
	LastUpdatedTime *metav1.Time `json:"lastUpdatedTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ArgoHelmRepoSecret is the Schema for the argohelmreposecrets API
type ArgoHelmRepoSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoHelmRepoSecretSpec   `json:"spec,omitempty"`
	Status ArgoHelmRepoSecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArgoHelmRepoSecretList contains a list of ArgoHelmRepoSecret
type ArgoHelmRepoSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoHelmRepoSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoHelmRepoSecret{}, &ArgoHelmRepoSecretList{})
}
