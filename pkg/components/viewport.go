package components

import (
	"github.com/ghodss/yaml"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
)

func GetNodeViewportContent(n *v1.Node) string {
	node := hideExtraneousNodeFields(n)
	return string(lo.Must(yaml.Marshal(node)))
}

func hideExtraneousNodeFields(n *v1.Node) *v1.Node {
	node := n.DeepCopy()
	node.ObjectMeta.ManagedFields = nil // We hid managedFields by default
	return node
}
