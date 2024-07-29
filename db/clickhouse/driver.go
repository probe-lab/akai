package clickhouse

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
)

const ()

type ConnectionDetails struct {
	Driver   string
	Address  string
	User     string
	Password string
	Database string
	Params   string
}

func (d ConnectionDetails) String() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s/%s%s",
		d.Driver,
		d.User,
		d.Password,
		d.Address,
		d.Database,
		d.Params,
	)
}

var DefaultDatabaseOpts = ConnectionDetails{
	Driver:   "clickhouse",
	Address:  "127.0.0.1:9000",
	User:     "default",
	Password: "",
	Database: "default",
	Params:   "",
}

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
	inputConverter func([]T) proto.Input
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

func (d tableDriver[T]) InputConverter() func([]T) proto.Input {
	return d.inputConverter
}
