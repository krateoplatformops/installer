package steps

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Add these result types to the existing file

type VarResult struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ObjectResult struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Operation  string `json:"operation"`
}

type ChartResult struct {
	ReleaseName  string      `json:"releaseName"`
	ChartName    string      `json:"chartName"`
	ChartVersion string      `json:"version"`
	AppVersion   string      `json:"appVersion"`
	Namespace    string      `json:"namespace"`
	Status       string      `json:"status"`
	Operation    string      `json:"operation"`
	Revision     int         `json:"revision,omitempty"`
	Updated      metav1.Time `json:"updated,omitempty"`
}
