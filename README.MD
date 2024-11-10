## Overview
TXTTunnel is a simple HTTP-based service for creating, sending, retrieving, and deleting text-based tunnels. It uses SSE (Server-Sent Events) for real-time communication between the client(s) and the server. Data sent to tunnels can either be sent via POST requests or through URL parameters.

## Endpoints

### Home Page
- **Endpoint:** `/`
- **Method:** `GET`
- **Description:** Displays the homepage of the TXTTunnel service.
- **Response:** 
    - `200 OK` with a welcome message.

### Create Tunnel
- **Endpoint:** `/api/v3/tunnel/create`
- **Methods:** `POST`, `GET`
- **Description:** Creates a new tunnel.
- **Request (POST):**
    - **Body:** JSON object containing the `id` field.
    ```json
    {
            "id": "tunnelId"
    }
    ```
- **Request (GET):**
    - **Query Parameters:** 
        - `id` (optional): If not provided, a random ID will be generated.
- **Response:**
    - `200 OK` with a JSON object containing the `id` of the created tunnel.
    ```json
    {
            "id": "tunnelId"
    }
    ```

### Stream Tunnel Content
- **Endpoint:** `/api/v3/tunnel/stream`
- **Methods:** `GET`, `POST`
- **Description:** Streams the content of a tunnel using Server-Sent Events (SSE).
- **Request (GET):**
    - **Query Parameters:**
        - `id`: The ID of the tunnel.
        - `subChannel` (optional): The subchannel to stream. Defaults to `main`.
- **Request (POST):**
    - **Body:** JSON object containing the `id` and `subChannel` fields.
    ```json
    {
            "id": "tunnelId",
            "subChannel": "subChannelName"
    }
    ```
- **Response:**
    - `200 OK` with SSE data.

### Get Tunnel Content
- **Endpoint:** `/api/v3/tunnel/get`
- **Methods:** `GET`, `POST`
- **Description:** Retrieves the content of a tunnel.
- **Request (GET):**
    - **Query Parameters:**
        - `id`: The ID of the tunnel.
        - `subChannel` (optional): The subchannel to retrieve. Defaults to `main`.
- **Request (POST):**
    - **Body:** JSON object containing the `id` and `subChannel` fields.
    ```json
    {
            "id": "tunnelId",
            "subChannel": "subChannelName"
    }
    ```
- **Response:**
    - `200 OK` with a JSON object containing the `content` of the specified subchannel.
    ```json
    {
            "content": "textData"
    }
    ```

### Send to Tunnel
- **Endpoint:** `/api/v3/tunnel/send`
- **Methods:** `POST`, `GET`
- **Description:** Sends data to a tunnel.
- **Request (POST):**
    - **Body:** JSON object containing the `id`, `subChannel`, and `content` fields.
    ```json
    {
            "id": "tunnelId",
            "subChannel": "subChannelName",
            "content": "textData"
    }
    ```
- **Request (GET):**
    - **Query Parameters:**
        - `id`: The ID of the tunnel.
        - `subChannel`: The subchannel to send data to.
        - `content`: The content to send.
- **Response:**
    - `200 OK` if the data is successfully sent.

## License
This project is licensed under the Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0) license. For more information, see the `LICENSE` file.
