package main

import (
	"encoding/json"
	"math/rand"
	"strconv"

	"github.com/gorilla/websocket"
)

type ReqCb func(res ResJson)
type ReqCbs map[string]ReqCb

type Client struct {
	ws  *websocket.Conn
	cbs ReqCbs
}

func makeClient() *Client {
	result := new(Client)
	result.cbs = make(ReqCbs)
	return result
}

func (c *Client) connect(urlStr string) error {
	ws, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		return err
	}

	c.ws = ws

	return nil
}

func (c *Client) sendReq(reqType string, rmsg json.RawMessage, cb ReqCb) error {

	Id := reqType +
		":" +
		strconv.Itoa(len(c.cbs)) +
		":" +
		strconv.Itoa(rand.Int())

	if cb != nil {
		c.cbs[Id] = cb
	}

	resJson := ReqJson{
		Id:   Id,
		Type: reqType,
		Msg:  rmsg,
	}
	c.ws.WriteJSON(resJson)
}
