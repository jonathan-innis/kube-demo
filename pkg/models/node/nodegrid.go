package node

import "github.com/bwagner5/kube-demo/pkg/models/grid"

func GridUpdate(m *grid.Model[Model, UpdateMsg, DeleteMsg], msg UpdateMsg) {
	if _, ok := m.Models[msg.GetID()]; !ok {
		m.Models[msg.GetID()] = NewModel(msg.Node)
	}
}

func GridDelete(m *grid.Model[Model, UpdateMsg, DeleteMsg], msg DeleteMsg) {
	delete(m.Models, msg.GetID())
}
