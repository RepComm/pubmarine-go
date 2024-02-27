package main

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Sub struct {
	ws *websocket.Conn
}

func makeSub(ws *websocket.Conn) *Sub {
	result := new(Sub)
	result.ws = ws
	return result
}

type Subs = map[*Sub]bool
type SubStorage = map[string]Subs

type WebSocketSubMap = map[*websocket.Conn]*Sub

type SchemaId string

type FieldId string
type FieldType int32

const (
	F_STR FieldType = iota
	F_INT
	F_FLOAT
	F_INT_ARRAY
	F_FLOAT_ARRAY
)

type Schema struct {
	Fields map[FieldId]FieldType
}
type Schemas = map[SchemaId]Schema

type InstanceData map[FieldId]any
type Instance struct {
	Data     InstanceData
	SchemaId SchemaId
}
type InstanceId string
type Instances = map[InstanceId]Instance

type ReqJson struct {
	Id   string
	Type string
	Msg  json.RawMessage
}
type ReqAuth struct {
	Id string
}
type ReqSub struct {
	Id      InstanceId
	IsTopic any
}
type ReqSchemaSet struct {
	SchemaId SchemaId
	Schema   Schema
}
type ReqSchemaGet struct {
	Id SchemaId
}

type ReqMut struct {
	Id     InstanceId
	Change InstanceData
}

type ReqInst struct {
	SchemaId SchemaId
}

type ResJson struct {
	Id    string
	Type  string
	Error any
	Msg   json.RawMessage
}
type ResList struct {
	Topics []SchemaId
}
type ResSchemaGet struct {
	Id     SchemaId
	Schema Schema
}
type ResPub struct {
	Id     InstanceId
	Change InstanceData
}

type ResInst struct {
	InstanceId InstanceId
	Instance   Instance
}

func FieldDefaultForType(FieldType FieldType) any {
	if FieldType == F_STR {
		return ""
	} else if FieldType == F_INT {
		return 0
	} else if FieldType == F_FLOAT {
		return 0.00
	} else if FieldType == F_FLOAT_ARRAY {
		return []float32{}
	} else if FieldType == F_INT_ARRAY {
		return []int32{}
	}
	return nil
}

func valueIsFieldType(v any, f FieldType, isStrict bool) bool {
	// fmt.Println("Type", reflect.TypeOf(v))

	switch v.(type) {
	case string:
		return f == F_STR
	case int:
	case float32:
	case float64:
		return f == F_FLOAT || f == F_INT
	}
	return false
}
