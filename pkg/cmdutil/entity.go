package cmdutil

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type HalkyonEntity interface {
	Get(string, v1.GetOptions) error
	Delete(string, *v1.DeleteOptions) error
	GetKnown() []string
	GetNamespace() string
}
