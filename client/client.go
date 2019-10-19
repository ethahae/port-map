package main

import (
	"flag"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/ethahae/port-map/util"
)

func makeConnection(targetTcpAddr, mapTcpAddr string) error {
	atomic.AddInt32(&idleChannels, 1)
	defer atomic.AddInt32(&idleChannels, -1)
	conn, err := net.Dial("tcp", mapTcpAddr)
	if err != nil {
		log.Printf("error when dail mapAddr %s, %s, try later", mapTcpAddr, err)
		return err
	} else {
		log.Printf("connect %s ok", mapTcpAddr)
	}
	data := make([]byte, 1)

	log.Printf("waiting remote connection trigger")
	if rCnt, err := conn.Read(data); err != nil {
		log.Printf("error when read %d, %s", rCnt, err)
		return err
	} else {
		log.Printf("read %d bytes, try init channel", rCnt)
	}
	//当代理端口有trigger数据时, 我们再去连接目标端口, 这样可以避免目标端口应用会检查超时
	rConn, err := net.Dial("tcp", targetTcpAddr)
	if err != nil {
		log.Printf("error when dail targetAddr %s,%s, try later", targetTcpAddr, err)
		return err
	} else {
		log.Printf("target connect ok. %s <-> %s", rConn.RemoteAddr().String(), conn.RemoteAddr().String())
	}
	go util.CopySocket(conn, rConn, &ongoingChannels)
	go util.CopySocket(rConn, conn, &ongoingChannels)
	log.Printf("connection channel init ok!")
	return nil
}

func constantMakeConnection(targetAddr, mapAddr string) {
	sleepSecond := int64(1)
	maxSpeepSecond := int64(60)
	for {
		if err := makeConnection(targetAddr, mapAddr); err != nil {
			sleepSecond = sleepSecond * 2
			if sleepSecond > maxSpeepSecond {
				sleepSecond = maxSpeepSecond
			}
			log.Printf("sleep %d second", sleepSecond)
			time.Sleep(time.Second * time.Duration(sleepSecond))
		}
	}
}

var idleChannels int32 = 0
var ongoingChannels int32 = 0

func main() {

	targetAddr := flag.String("target", "172.17.2.10:3307", "要映射的目标地址")
	mapAddr := flag.String("server", "172.17.32.52:8811", "公网服务器监听地址")
	channel := flag.Int("channel", 3, "并发代理链接数量")
	flag.Parse()
	log.Printf("map %s to %s with %d channels", *targetAddr, *mapAddr, *channel)
	for i := 0; i < *channel; i++ {
		go constantMakeConnection(*targetAddr, *mapAddr)
	}
	for {
		select {
		case <-time.After(time.Second * 4):
			log.Printf("idle %d, ongoing %d", atomic.LoadInt32(&idleChannels), atomic.LoadInt32(&ongoingChannels)/2)
		}
	}

}
