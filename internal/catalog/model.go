package catalog

import "errors"

var ErrNotFound = errors.New("node not found")

type Node struct {
	ID       string
	ParentID *string
	Name     string
}

type Component struct {
	ID     string
	NodeID string
	Name   string
}

type Product struct {
	ID     string
	NodeID string
	Name   string
}
