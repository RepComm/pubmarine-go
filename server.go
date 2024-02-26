package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type Server struct {
	wsUpgrader       websocket.Upgrader
	wsSubMap         WebSocketSubMap
	topicSubscribers SubStorage
	idSubscribers    SubStorage
	schemas          Schemas
	instances        Instances
}

func (s *Server) handle_sub_msg(sub *Sub, raw []byte, msgType int) {
	if msgType != 1 {
		return
	}

	req := &ReqJson{}

	err := json.Unmarshal(raw, &req)
	if err != nil {
		fmt.Println(err)
		return
	}
	if req.Type == "auth" {
		msg := &ReqAuth{}

		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(msg)
	} else if req.Type == "sub" {
		msg := &ReqSub{}
		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}

		isTopic := msg.IsTopic == true

		s.setSubscribe(msg.Id, isTopic, sub, true)
	} else if req.Type == "unsub" {
		msg := &ReqSub{}
		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}

		isTopic := msg.IsTopic == true

		s.setSubscribe(msg.Id, isTopic, sub, false)
	} else if req.Type == "list" {
		rmsg := &ResList{
			Topics: []string{},
		}
		for k, _ := range s.schemas {
			rmsg.Topics = append(rmsg.Topics, k)
		}

		Msg, err := json.Marshal(rmsg)
		if err != nil {
			fmt.Println("list rmsg marshal", err)
			return
		}

		res := &ResJson{
			Id:   req.Id,
			Type: req.Type,
			Msg:  Msg,
		}

		sub.ws.WriteJSON(res)
	} else if req.Type == "schema-set" {
		msg := &ReqSchemaSet{}

		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}

		s.schemas[msg.Id] = msg.Change
	} else if req.Type == "schema-get" {
		msg := &ReqSchemaGet{}

		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}

		rmsg := &ResSchemaGet{
			Id:     msg.Id,
			Schema: s.schemas[msg.Id],
		}

		Msg, err := json.Marshal(rmsg)
		if err != nil {
			fmt.Println("list rmsg marshal", err)
			return
		}

		res := &ResJson{
			Id:   req.Id,
			Type: req.Type,
			Msg:  Msg,
		}

		sub.ws.WriteJSON(res)
	} else if req.Type == "mut" {
		msg := &ReqMut{}
		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		inst, ok := s.instances[msg.Id]
		if ok {
			for k, _ := range inst {
				inst[k] = msg.Change[k]
			}
			s.instances[msg.Id] = inst
		} else {
			inst = msg.Change
			s.instances[msg.Id] = inst
		}

		resmsg := &ResPub{
			Id:       msg.Id,
			Instance: inst,
		}
		ResMsg, err := json.Marshal(resmsg)
		if err != nil {
			fmt.Println("list rmsg marshal", err)
			return
		}

		res := &ResJson{
			Id:   req.Id,
			Type: req.Type,
			Msg:  ResMsg,
		}

		Res, err := json.Marshal(res)
		if err != nil {
			fmt.Println("RMsg marshal", err)
			return
		}

		s.walkSubs(msg.Id, false, func(sub *Sub) {
			sub.ws.WriteMessage(websocket.TextMessage, Res)
		})

	} else {
		fmt.Println("Unhandled json Type:", req.Type)
	}
}

func (s *Server) handle_http(w http.ResponseWriter, r *http.Request) {
	c, err := s.wsUpgrader.Upgrade(w, r, nil)

	if err != nil {
		// log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	s.wsSubMap[c] = makeSub(c)

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {

			delete(s.wsSubMap, c)

			// log.Println("read:", err)
			break
		}

		s.handle_sub_msg(s.wsSubMap[c], message, mt)
	}
}

func makeServer() *Server {
	result := new(Server)

	result.wsSubMap = make(map[*websocket.Conn]*Sub)

	result.topicSubscribers = make(SubStorage)
	result.idSubscribers = make(SubStorage)

	result.schemas = make(Schemas)

	result.instances = make(Instances)

	return result
}

func (s *Server) subsGetOrCreate(topicOrId string, isTopic bool) Subs {
	var result Subs = nil
	if isTopic {
		result = s.topicSubscribers[topicOrId]
	} else {
		result = s.idSubscribers[topicOrId]
	}

	if result == nil {
		result = make(Subs)
		if isTopic {
			s.topicSubscribers[topicOrId] = result
		} else {
			s.idSubscribers[topicOrId] = result
		}
	}

	return result
}

func (s *Server) setSubscribe(topicOrId string, isTopic bool, sub *Sub, isSub bool) {
	subs := s.subsGetOrCreate(topicOrId, isTopic)
	if isSub {
		subs[sub] = true
	} else {
		delete(subs, sub)
	}
	// fmt.Println(topicOrId, isTopic, *sub, isSub)
}

type walkSubsCb = func(sub *Sub)

func (s *Server) walkSubs(topicOrId string, isTopic bool, cb walkSubsCb) {
	var subs Subs = nil
	if isTopic {
		subs = s.subsGetOrCreate(topicOrId, isTopic)
	} else {
		subs = s.subsGetOrCreate(topicOrId, isTopic)
	}
	for k, _ := range subs {
		cb(k)
	}
}

func (s *Server) init(addr string) {
	s.wsUpgrader = websocket.Upgrader{}

	//allow cross origin
	s.wsUpgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	http.HandleFunc("/", s.handle_http)
	http.ListenAndServe(addr, nil)
}
