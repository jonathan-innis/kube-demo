package node

import v1 "k8s.io/api/core/v1"

func GetCondition(n *v1.Node, match v1.NodeConditionType) v1.NodeCondition {
	for _, condition := range n.Status.Conditions {
		if condition.Type == match {
			return condition
		}
	}
	return v1.NodeCondition{}
}
