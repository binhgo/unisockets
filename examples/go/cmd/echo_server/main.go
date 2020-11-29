package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/pojntfx/webassembly-berkeley-sockets-via-webrtc/examples/go/pkg/sockets"
)

var (
	LOCAL_HOST = []byte{10, 0, 0, 240}
)

const (
	LOCAL_PORT = 1234
	BACKLOG    = 1

	BUFFER_LENGTH = 1024
)

func main() {
	// Create address
	serverAddress := sockets.SockaddrIn{
		SinFamily: sockets.PF_INET,
		SinPort:   sockets.Htons(LOCAL_PORT),
		SinAddr: struct{ SAddr uint32 }{
			SAddr: binary.LittleEndian.Uint32(LOCAL_HOST),
		},
	}
	serverAddressReadable := fmt.Sprintf("%v:%v", LOCAL_HOST, LOCAL_PORT)

	// Create socket
	serverSocket, err := sockets.Socket(sockets.PF_INET, sockets.SOCK_STREAM, 0)
	if err != nil {
		fmt.Printf("[ERROR] Could not create socket %v: %v\n", serverAddressReadable, err)

		os.Exit(1)
	}

	// Bind
	if err := sockets.Bind(serverSocket, &serverAddress); err != nil {

		os.Exit(1)
	}

	// Listen
	if err := sockets.Listen(serverSocket, BACKLOG); err != nil {
		fmt.Printf("[ERROR] Could not listen on socket %v: %v\n", serverAddressReadable, err)

		os.Exit(1)
	}

	fmt.Println("[INFO] Listening on", serverAddressReadable)

	// Accept loop
	for {
		fmt.Println("[DEBUG] Accepting on", serverAddressReadable)

		clientAddress := sockets.SockaddrIn{}

		// Accept
		clientSocket, err := sockets.Accept(serverSocket, &clientAddress)
		if err != nil {
			fmt.Println("[ERROR] Could not accept, continuing:", err)

			continue
		}

		go func(innerClientSocket int32, innerClientAddress sockets.SockaddrIn) {
			clientHost := make([]byte, 4) // xxx.xxx.xxx.xxx
			binary.LittleEndian.PutUint32(clientHost, uint32(innerClientAddress.SinAddr.SAddr))

			clientAddressReadable := fmt.Sprintf("%v:%v", clientHost, innerClientAddress.SinPort)

			fmt.Println("[INFO] Accepted client", clientAddressReadable)

			// Receive loop
			for {
				fmt.Printf("[DEBUG] Waiting for client %v to send\n", clientAddressReadable)

				// Receive
				receivedMessage := make([]byte, BUFFER_LENGTH)

				receivedMessageLength, err := sockets.Recv(innerClientSocket, &receivedMessage, BUFFER_LENGTH, 0)
				if err != nil {
					fmt.Printf("[ERROR] Could not receive from client %v, dropping message: %v\n", clientAddressReadable, err)

					continue
				}

				if receivedMessageLength == 0 {
					break
				}

				fmt.Printf("[DEBUG] Received %v bytes from %v\n", receivedMessageLength, clientAddressReadable)

				// Send
				sentMessage := []byte(fmt.Sprintf("You've sent: %v", string(receivedMessage)))

				sentMessageLength, err := sockets.Send(innerClientSocket, sentMessage, 0)
				if err != nil {
					fmt.Printf("[ERROR] Could not send to client %v, dropping message: %v\n", clientAddressReadable, err)

					break
				}

				fmt.Printf("[DEBUG] Sent %v bytes to %v\n", sentMessageLength, clientAddressReadable)
			}

			fmt.Println("[INFO] Disconnected from client", clientAddressReadable)

			// Shutdown
			if err := sockets.Shutdown(innerClientSocket, sockets.SHUT_RDWR); err != nil {
				fmt.Printf("[ERROR] Could not shutdown client socket %v, continuing: %v\n", clientAddressReadable, err)
			}
		}(clientSocket, clientAddress)
	}
}