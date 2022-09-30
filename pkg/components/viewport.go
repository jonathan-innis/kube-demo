package components

import (
	"github.com/ghodss/yaml"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MarshalYAML(obj client.Object) string {
	objCopy := obj.DeepCopyObject().(client.Object)
	objCopy.SetManagedFields(nil) // We hid managedFields by default
	return string(lo.Must(yaml.Marshal(objCopy)))
}

func MarshalJSON(obj client.Object) string {
	objCopy := obj.DeepCopyObject().(client.Object)
	objCopy.SetManagedFields(nil) // We hid managedFields by default
	return string(lo.Must(json.Marshal(objCopy)))
}
