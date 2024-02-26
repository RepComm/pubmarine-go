# pubmarine-go

pubmarine is a pub/sub server and client SDK set for

- js
  - web client - websocket
  - node.js server - websocket / UDP
- lua
  - minetest client - UDP
- go
  - server

## TODO before MVP
- authentication
- go client impl
- update js/lua impl to match go, some differencnes exist right now
- go UDP support
- instance array prop partial update / partial subscribe (like Uint8Array without sending entire array upon mutation)
