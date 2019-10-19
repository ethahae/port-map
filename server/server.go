package main

import (
	"flag"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/ethahae/port-map/util"
)

func acceptAll(listener net.Listener, inputChan chan net.Conn) {
	for {
		conn, err := listener.Accept()
		if err == nil {
			inputChan <- conn
		} else {
			log.Printf("erroe when accept %s, %s", listener.Addr().String(), err)
		}
	}
}

func checkConnectionClose(conn net.Conn, grab chan net.Conn) {
	defer atomic.AddInt32(&idleProxyConnection, -1)
	for {
		select {
		case grab <- conn:
			log.Printf("connection grab ")
			return
		default:
			data := make([]byte, 1)
			conn.SetDeadline(time.Now().Add(time.Millisecond * 10))
			if cnt, err := conn.Read(data); err == io.EOF {
				log.Printf(" proxy  connection closed %s,%d,%s", conn.RemoteAddr().String(), cnt, err)
				return
			}
		}
	}

}

func makePair(localConn net.Conn, grab chan net.Conn) {
	select {
	case proxyConn := <-grab:
		proxyConn.SetDeadline(time.Now().Add(time.Hour * 7 * 24))
		trigger := make([]byte, 1)
		log.Printf("make a pair %s <-> %s <-> %s <-> %s",
			proxyConn.RemoteAddr().String(), proxyConn.LocalAddr().String(),
			localConn.LocalAddr().String(), localConn.RemoteAddr().String())
		cnt, err := proxyConn.Write(trigger)
		if err != nil {
			log.Printf("write trigger fails, %d, %v", cnt, err)
		} else {
			log.Printf("channel start after write %d", cnt)
			go util.CopySocket(proxyConn, localConn, &ongoingProxyConnection)
			go util.CopySocket(localConn, proxyConn, &ongoingProxyConnection)
		}
	case <-time.After(time.Second * 2):
		log.Printf("time out, close conn")
		localConn.Close()
	}
}

var idleProxyConnection int32 = 0
var ongoingProxyConnection int32 = 0

func main() {

	proxyListenAddr := flag.String("proxy", ":8811", "port-map client 链接用的地址")
	localListenAddr := flag.String("local", ":8812", "本地应用连接用的地址")
	flag.Parse()

	proxyListener, err := net.Listen("tcp", *proxyListenAddr)
	if err != nil {
		log.Fatalf("listen %s, %s", *proxyListenAddr, err)
	} else {
		log.Printf("listen on %s", *proxyListenAddr)
	}
	localListener, err := net.Listen("tcp", *localListenAddr)
	if err != nil {
		log.Fatalf("listen %s, %s", *localListenAddr, err)
	} else {
		log.Printf("listen on %s", *localListenAddr)
	}

	newProxyConnectionChan := make(chan net.Conn)
	proxyConnectionGrabChan := make(chan net.Conn)
	newLocalConnectionChan := make(chan net.Conn)

	go acceptAll(proxyListener, newProxyConnectionChan)
	go acceptAll(localListener, newLocalConnectionChan)

	for {
		select {
		case proxyConn := <-newProxyConnectionChan: //our clients try connect
			atomic.AddInt32(&idleProxyConnection, 1)
			log.Printf("got proxy connection %s", proxyConn.RemoteAddr().String())
			go checkConnectionClose(proxyConn, proxyConnectionGrabChan)
		case localConn := <-newLocalConnectionChan: //local  app try connect
			log.Printf("got local connection %s", localConn.RemoteAddr().String())
			go makePair(localConn, proxyConnectionGrabChan)
		case <-time.After(time.Second * 2):
			log.Printf("idle %d, ongoing %d",
				atomic.LoadInt32(&idleProxyConnection),
				atomic.LoadInt32(&ongoingProxyConnection)/2)
		}

	}

}
