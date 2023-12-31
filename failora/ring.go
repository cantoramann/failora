package main

// var singletonRing *Ring = nil

// type Ring struct {
// 	Nodes []*NetworkNode
// }

// func NewRing() *Ring {
// 	return &Ring{}
// }

// func GetRing() *Ring {
// 	if singletonRing == nil {
// 		singletonRing = NewRing()
// 	}
// 	return singletonRing
// }

// func (r *Ring) AddNode(node *NetworkNode) {
// 	if len(r.Nodes) > 0 {
// 		lastNode := r.Nodes[len(r.Nodes)-1]
// 		lastNode.successor = node
// 	}
// 	r.Nodes = append(r.Nodes, node)
// 	node.successor = r.Nodes[0]
// }
