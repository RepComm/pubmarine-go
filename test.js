let ws = new WebSocket("ws://localhost:10209")

ws.onopen = () => {

  ws.onmessage = (evt) => {
    let msg = {}
    try {
      msg = JSON.parse(evt.data);
    } catch (ex) {
      console.warn(ex);
      return;
    }
    if (msg.Type === undefined || msg.Id === undefined) {
      console.warn("msg.Type or msg.Id missing:", msg);
      return;
    }
    console.log(msg);

    let cb = cbs.get(msg.Id);
    if (cb) {
      cb(msg);
      cbs.delete(msg.Id);
    }
  }
  let cbs = new Map();
  let sendJson = (obj) => ws.send(JSON.stringify(obj));
  let sendReq = (type, rmsg, cb)=> {
    const Id = `${type}:${cbs.size}:${Math.floor(Math.random()*Number.MAX_SAFE_INTEGER)}`;
    
    if (cb) {
      cbs.set(Id, cb);
    }

    sendJson({
      Id,
      Type: type,
      Msg: rmsg
    })
  };

  //create a schema
  const schemaId = "test";

  sendReq("schema-set", {
    SchemaId: schemaId,
    Schema: {
      Fields: {
        A: 1,
        B: 1
      }
    }
  })

  sendReq("inst", {
    SchemaId: schemaId
  }, (msg)=>{

    const InstanceId = msg.Msg.InstanceId;
  
    // subscribe to an instance by id
    sendReq("sub", {
      Id: InstanceId
    });
    
    //send a mutation
    sendReq("mut", {
      Id: InstanceId,
      Change: {
        A: 40
      }
    });

  });

}

