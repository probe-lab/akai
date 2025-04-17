package clickhouse

import (
	"github.com/ClickHouse/ch-go/proto"
)

type Persistable interface {
	TableName() string
	QueryValues() map[string]any
}

type connectableTable interface {
	Tag() string
	TableName() string
}

var _ connectableTable = (*tableDriver[interface{}])(nil)

type tableDriver[T any] struct {
	tableName      string
	tag            string
	baseQuery      string
	inputConverter func([]*T) proto.Input
}

func (d tableDriver[T]) TableName() string {
	return d.tableName
}

func (d tableDriver[T]) Tag() string {
	return d.tag
}

func (d tableDriver[T]) BaseQuery() string {
	return d.baseQuery
}

func (d tableDriver[T]) InputConverter() func([]*T) proto.Input {
	return d.inputConverter
}
