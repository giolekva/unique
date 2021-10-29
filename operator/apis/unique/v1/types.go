package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CountUnique struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   CountUniqueSpec   `json:"spec"`
	Status CountUniqueStatus `json:"status,omitempty"`
}

type CountUniqueSpec struct {
	StartFrom    string `json:"startFrom"`
	NumDocuments int    `json:"numDocuments"`
	NumBits      int    `json:"numBits"`
}

type CountUniqueStatus struct {
	State   CountUniqueState `json:"state,omitempty"`
	Message string           `json:"message,omitempty"`
}

type CountUniqueState string

const (
	CountUniqueStateRunning CountUniqueState = "Running"
	CountUniqueStateDone    CountUniqueState = "Done"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CountUniqueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CountUnique `json:"items"`
}
