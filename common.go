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

type Schema struct {
}
type Schemas = map[string]Schema

type Instance = map[string]any
type Instances = map[string]Instance

type ReqJson struct {
	Id   string
	Type string
	Msg  json.RawMessage
}
type ReqAuth struct {
	Id string
}
type ReqSub struct {
	Id      string
	IsTopic any
}
type ReqSchemaSet struct {
	Id     string
	Change Schema
}
type ReqSchemaGet struct {
	Id string
}

type ReqMut struct {
	Id     string
	Change Instance
}

type ResJson struct {
	Id   string
	Type string
	Msg  json.RawMessage
}
type ResList struct {
	Topics []string
}
type ResSchemaGet struct {
	Id     string
	Schema Schema
}
type ResPub struct {
	Id       string
	Instance Instance
}
