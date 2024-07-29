package db

type Persistable interface {
	TableName() string
	QueryValues() map[string]any
}

var _ Persistable = (*Network)(nil)
