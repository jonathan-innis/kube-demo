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

type Status string

const (
	Unknown  Status = "Unknown"
	Ready    Status = "Ready"
	NotReady Status = "NotReady"
	Cordoned Status = "Cordoned"
)

func GetReadyStatus(node *v1.Node) Status {
	switch {
	case node.Spec.Unschedulable:
		return Cordoned
	case GetCondition(node, v1.NodeReady).Status == v1.ConditionTrue:
		return Ready
	case GetCondition(node, v1.NodeReady).Status == v1.ConditionFalse:
		return NotReady
	default:
		return Unknown
	}
}
