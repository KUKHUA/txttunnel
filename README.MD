# TXTTunnel
TXTTunnel is a simple HTTP-based service for creating, sending, retrieving, and deleting text-based tunnels. It uses BadgerDB for in-memory storage and provides an API for interaction.

## Features

- Create a new tunnel with a unique ID and authentication token.
- Send data to a tunnel.
- Retrieve data from a tunnel.
- Delete a tunnel using an authentication token.
- Automatic cleanup of expired tunnels.

## Endpoints

### Home Page

- **URL:** `/`
- **Method:** `GET`
- **Description:** Displays a welcome message.

### Create Tunnel

- **URL:** `/tunnel/create`
- **Method:** `POST`
- **Description:** Creates a new tunnel and returns the tunnel ID and authentication token for deleting the tunnel.
- **Response:**
    ```json
    {
        "id": "tunnelId",
        "auth": "authToken"
    }
    ```

### Send to Tunnel

- **URL:** `/tunnel/send`
- **Method:** `POST`
- **Query Parameters:**
    - `id`: The tunnel ID.
    - `content`: The content to send to the tunnel.
- **Description:** Sends data to a tunnel.

### Get Tunnel

- **URL:** `/tunnel`
- **Method:** `GET`
- **Query Parameters:**
    - `id`: The tunnel ID.
- **Description:** Retrieves data from a tunnel.
- **Response:**
    ```json
    {
        "id": "tunnelId",
        "content": "tunnelContent"
    }
    ```

### Delete Tunnel

- **URL:** `/tunnel/delete`
- **Method:** `DELETE`
- **Query Parameters:**
    - `id`: The tunnel ID.
    - `auth`: The authentication token.
- **Description:** Deletes a tunnel using the authentication token.

## Setup

1. Clone the repository.
2. Install dependencies using `go get`.
3. Run the server using `go run main.go`.

## Configuration

- `timeToLive`: Time to live for each tunnel in minutes (default: 5 minutes).
- `cleanUpInterval`: Interval for cleaning up expired tunnels in minutes (default: 10 minutes).

## License

This project is licensed under Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0) license. For more information, see [the file](LICENSE).
