package util

import (
	"io"
	"log"
	"net"
	"sync/atomic"
)

func CopySocket(writer, reader net.Conn, counter *int32) {
	defer writer.Close()
	defer reader.Close()
	defer atomic.AddInt32(counter, -1)
	atomic.AddInt32(counter, 1)
	cnt, err := io.Copy(writer, reader)
	log.Printf("channel %s <-> %s <-> %s <-> %s, cnt %d, error %s",
		reader.RemoteAddr().String(), reader.LocalAddr().String(),
		writer.LocalAddr().String(), writer.RemoteAddr().String(),
		cnt, err)
}
