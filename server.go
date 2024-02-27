package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"

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

func respond(sub *Sub, req *ReqJson, msg json.RawMessage) {
	res := &ResJson{
		Type: req.Type,
		Id:   req.Id,
		Msg:  msg,
	}
	sub.ws.WriteJSON(res)
}
func respondError(sub *Sub, req *ReqJson, errorString string) {
	res := &ResJson{
		Type:  req.Type,
		Id:    req.Id,
		Msg:   nil,
		Error: errorString,
	}
	sub.ws.WriteJSON(res)
}

func calcUniqueInstanceId(schemaId SchemaId, count int) InstanceId {
	str := string(schemaId) + ":" + strconv.Itoa(count)

	h := fnv.New32a()
	h.Write([]byte(str))
	hash := int(h.Sum32())

	return InstanceId(strconv.Itoa(hash))
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
			Topics: []SchemaId{},
		}
		for k := range s.schemas {
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

		s.schemas[msg.SchemaId] = msg.Schema
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
		//what instance are we talking about?
		instance, isValidInstanceId := s.instances[msg.Id]

		//make sure an instance already exists with the id for mutation
		if !isValidInstanceId {
			//if instance doesn't exist, don't try to apply changes
			return
		}

		//get the schema for this instance
		// instanceSchema := s.schemas[msg.Change.SchemaId]
		instanceSchema := s.schemas[instance.SchemaId]

		//get the change
		Change := msg.Change

		AppliedChanges := map[FieldId]any{}

		//loop over each key in Change Data
		for ChangeFieldId, ChangeFieldValue := range Change {

			//get the field type
			SchemaFieldType, isValidField := instanceSchema.Fields[ChangeFieldId]

			// fmt.Println(instance, ChangeFieldId, ChangeFieldValue)

			//schema must have field, and it's type must match the change data
			if isValidField {
				if valueIsFieldType(ChangeFieldValue, SchemaFieldType, false) {
					//apply the change of this field to the instance
					AppliedChanges[ChangeFieldId] = ChangeFieldValue

					instance.Data[ChangeFieldId] = ChangeFieldValue

					// fmt.Println("Applied", instance, ChangeFieldId, ChangeFieldValue)
				} else {
					fmt.Println("invalid field value", ChangeFieldValue, SchemaFieldType)
				}
			} else {
				fmt.Println("invalid field", ChangeFieldId)
			}
		}
		//re-apply the instance to the store (we're not using pointers)
		s.instances[msg.Id] = instance

		//push changes to subscribers

		resmsg := &ResPub{
			Id:     msg.Id,
			Change: AppliedChanges,
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
	} else if req.Type == "inst" {
		msg := &ReqInst{}
		err = json.Unmarshal(req.Msg, &msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		schema, isValidSchema := s.schemas[msg.SchemaId]
		if !isValidSchema {
			respondError(sub, req, "unknown schema, cannot instance")
			return
		}
		Data := InstanceData{}

		for FieldId, FieldType := range schema.Fields {
			Data[FieldId] = FieldDefaultForType(FieldType)
		}

		Instance := Instance{
			SchemaId: msg.SchemaId,
			Data:     Data,
		}

		count := len(s.instances)
		InstanceId := calcUniqueInstanceId(msg.SchemaId, count)

		resinst := &ResInst{
			Instance:   Instance,
			InstanceId: InstanceId,
		}

		Msg, err := json.Marshal(resinst)
		if err != nil {
			respondError(sub, req, "server error: couldn't marshal instance json")
			return
		}
		respond(sub, req, Msg)
		s.instances[InstanceId] = Instance

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

func (s *Server) subsGetOrCreate(topicOrId InstanceId, isTopic bool) Subs {
	var result Subs = nil
	if isTopic {
		result = s.topicSubscribers[string(topicOrId)]
	} else {
		result = s.idSubscribers[string(topicOrId)]
	}

	if result == nil {
		result = make(Subs)
		if isTopic {
			s.topicSubscribers[string(topicOrId)] = result
		} else {
			s.idSubscribers[string(topicOrId)] = result
		}
	}

	return result
}

func (s *Server) setSubscribe(topicOrId InstanceId, isTopic bool, sub *Sub, isSub bool) {
	subs := s.subsGetOrCreate(topicOrId, isTopic)
	if isSub {
		subs[sub] = true
	} else {
		delete(subs, sub)
	}
	// fmt.Println(topicOrId, isTopic, *sub, isSub)
}

type walkSubsCb = func(sub *Sub)

func (s *Server) walkSubs(topicOrId InstanceId, isTopic bool, cb walkSubsCb) {
	var subs Subs = nil
	if isTopic {
		subs = s.subsGetOrCreate(topicOrId, isTopic)
	} else {
		subs = s.subsGetOrCreate(topicOrId, isTopic)
	}
	for k := range subs {
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
