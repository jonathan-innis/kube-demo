package node

import "github.com/bwagner5/kube-demo/pkg/models/grid"

func GridUpdate(m *grid.Model[*Model, UpdateMsg, DeleteMsg], msg UpdateMsg) {
	if _, ok := m.Models.Get(msg.GetID()); !ok {
		m.Models.Load(msg.GetID(), NewModel(msg.Node))
	}
}

func GridDelete(m *grid.Model[*Model, UpdateMsg, DeleteMsg], msg DeleteMsg) {
	m.Models.Delete(msg.GetID())
}
