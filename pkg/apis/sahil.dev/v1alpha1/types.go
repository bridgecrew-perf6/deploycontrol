package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type HADeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HADeploymentSpec `json:"spec"`
}

type HADeploymentSpec struct {
	Replicas int32  `json:"replicas"`
	Image    string `json:"image"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type HADeploymentList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ListMeta   `json:"metadata"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Items []HADeployment `json:"items"`
}
