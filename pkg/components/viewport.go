package components

import (
	"github.com/ghodss/yaml"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MarshalViewportContent(obj client.Object) string {
	objCopy := obj.DeepCopyObject().(client.Object)
	objCopy.SetManagedFields(nil) // We hid managedFields by default
	return string(lo.Must(yaml.Marshal(objCopy)))
}
