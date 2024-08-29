package db

import "github.com/probe-lab/akai/db/models"

type Persistable interface {
	TableName() string
	QueryValues() map[string]any
}

var _ Persistable = (*models.Network)(nil)
