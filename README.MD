# TXTTunnel
TXTTunnel is a simple HTTP-based service for creating, sending, retrieving, and deleting text-based tunnels. It uses SSE (Server-Sent Events) for real-time communication between the client(s) and the server. Data sent to tunnels can either be sent via POST requests or through url parameters.

## Endpoints
 - `/` - The home page of the service. 
 - `/tunnel/stream?id=tunnelId` - The endpoint for streaming Server-Sent Events. The `tunnelId` parameter is the ID of the tunnel to stream.
 - `/tunnel/create` - The endpoint for creating a new tunnel. It will return the ID of the newly created tunnel in JSON.
 - `/tunnel/send?id=tunnelId&content=textData` - The endpoint for sending data to a tunnel. The `tunnelId` parameter is the ID of the tunnel to send data to. The content of the data is specified in the `content` parameter. This is the GET method.
 - `/tunnel/send/post` - The endpoint for sending data to a tunnel via POST requests. Both the tunnel ID and the content of the data are specified in the POST request body. For example, the body of the POST request should be in JSON format:
 ```json
 {
     "id": "tunnelId",
     "content": "textData"
 }
```

## License

This project is licensed under Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0) license. For more information, see [the file](LICENSE).
