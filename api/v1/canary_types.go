/*


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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CanarySpec defines the desired state of Canary
type CanarySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Size            int32              `json:"size"`
	KafkaConfig     KafkaClusterConfig `json:"kafkaConfig"`
	TrafficProducer TrafficProducer    `json:"trafficProducer,omitempty"`
	SecretVolumes   []SecretVolume     `json:"secretVolumes,omitempty"`
}

type KafkaClusterConfig struct {
	BootstrapUrl string `json:"bootstrapUrl"`
}

type TrafficProducer struct {
	Topic    string `json:"topic"`
	SendRate string `json:"sendRate"`
	ClientId string `json:"clientId"`
}

type SecretVolume struct {
	Name       string `json:"name"`
	MountPath  string `json:"mountPath"`
	SecretName string `json:"secretName"`
}

// CanaryStatus defines the observed state of Canary
type CanaryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Nodes []string `json:"nodes"`
}

// +kubebuilder:object:root=true

// Canary is the Schema for the canaries API
// +kubebuilder:subresource:status
type Canary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CanarySpec   `json:"spec,omitempty"`
	Status CanaryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CanaryList contains a list of Canary
type CanaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Canary `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Canary{}, &CanaryList{})
}
