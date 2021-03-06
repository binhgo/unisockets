// +build !js,tinygo

package unisockets

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

const (
	PF_INET     = uint16(C.PF_INET)
	SOCK_STREAM = int32(C.SOCK_STREAM)
	SHUT_RDWR   = int32(C.SHUT_RDWR)
)

func Socket(socketDomain uint16, socketType int32, socketProtocol int32) (int32, error) {
	rv := int32(socket(C.int(socketDomain), C.int(socketType), C.int(socketProtocol)))
	if rv == -1 {
		return rv, fmt.Errorf("could not create socket, error code %v", rv)
	}

	return rv, nil
}

func Bind(socketFd int32, socketAddr *SockaddrIn) error {
	addr := C.sockaddr_in{
		sin_family: C.ushort(socketAddr.SinFamily),
		sin_port:   uint16(socketAddr.SinPort),
		sin_addr: C.in_addr{
			s_addr: uint32(socketAddr.SinAddr.SAddr),
		},
	}

	if rv := int32(bind(C.int(socketFd), (*C.sockaddr)(unsafe.Pointer(&addr)), C.uint(unsafe.Sizeof(addr)))); rv == -1 {
		return fmt.Errorf("could not bind socket, error code %v", rv)
	}

	return nil
}

func Listen(socketFd int32, socketBacklog int32) error {
	if rv := int32(listen(C.int(socketFd), C.int(socketBacklog))); rv == -1 {
		return fmt.Errorf("could not listen on socket, error code %v", rv)
	}

	return nil
}

func Accept(socketFd int32, socketAddr *SockaddrIn) (int32, error) {
	addr := C.sockaddr_in{
		sin_family: C.ushort(socketAddr.SinFamily),
		sin_port:   uint16(socketAddr.SinPort),
		sin_addr: C.in_addr{
			s_addr: uint32(socketAddr.SinAddr.SAddr),
		},
	}

	addrLen := C.uint(unsafe.Sizeof(socketAddr))

	rv := int32(accept(C.int(socketFd), (*C.sockaddr)(unsafe.Pointer(&addr)), &addrLen))
	if rv == -1 {
		return rv, fmt.Errorf("could not accept on socket, error code %v", rv)
	}

	socketAddr.SinFamily = uint16(addr.sin_family)
	socketAddr.SinPort = uint16(addr.sin_port)
	socketAddr.SinAddr.SAddr = uint32(addr.sin_addr.s_addr)

	return rv, nil
}

func Recv(socketFd int32, socketReceivedMessage *[]byte, socketBufferLength uint32, socketFlags int32) (int32, error) {
	receivedMessage := make([]byte, socketBufferLength)

	rv := int32(recv(C.int(socketFd), unsafe.Pointer(&receivedMessage[0]), C.ulong(socketBufferLength), C.int(socketFlags)))
	if rv == -1 {
		return rv, fmt.Errorf("could not receive from socket, error code %v", rv)
	}

	outReceivedMessage := []byte(receivedMessage)

	*socketReceivedMessage = outReceivedMessage

	return rv, nil
}

func Send(socketFd int32, socketMessageToSend []byte, socketFlags int32) (int32, error) {
	rv := int32(send(C.int(socketFd), unsafe.Pointer(&socketMessageToSend[0]), C.ulong(len(socketMessageToSend)), C.int(socketFlags)))
	if rv == -1 {
		return rv, fmt.Errorf("could not send from socket, error code %v", rv)
	}

	return rv, nil
}

func Shutdown(socketFd int32, socketFlags int32) error {
	if rv := shutdown(C.int(socketFd), C.int(socketFlags)); rv == -1 {
		return fmt.Errorf("could not shut down socket, error code %v", rv)
	}

	return nil
}

func Connect(socketFd int32, socketAddr *SockaddrIn) error {
	addr := C.sockaddr_in{
		sin_family: C.ushort(socketAddr.SinFamily),
		sin_port:   uint16(socketAddr.SinPort),
		sin_addr: C.in_addr{
			s_addr: uint32(socketAddr.SinAddr.SAddr),
		},
	}

	if rv := connect(C.int(socketFd), (*C.sockaddr)(unsafe.Pointer(&addr)), C.uint(unsafe.Sizeof(addr))); rv == -1 {
		return fmt.Errorf("could not connect to socket, error code %v", rv)
	}

	return nil
}

func Htons(v uint16) uint16 {
	return uint16(C.htons(v))
}
