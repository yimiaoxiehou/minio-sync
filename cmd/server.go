package cmd

import (
	"fmt"
	"sync/atomic"

	"github.com/panjf2000/gnet/v2"
	"google.golang.org/protobuf/proto"

	"log"

	"github.com/yimiaoxiehou/minio-sync/internal/message"
	"github.com/yimiaoxiehou/minio-sync/internal/minio"
	"github.com/yimiaoxiehou/minio-sync/internal/protocol"
)

// main is the entry point of the program
type server struct {
	gnet.BuiltinEventEngine
	eng       gnet.Engine
	network   string
	addr      string
	multicore bool
	connected int32
}

var dataBuffer chan ([]byte) = make(chan ([]byte))

// var respBuffer chan (*message.RespMessage) = make(chan (*message.RespMessage))

// OnBoot description of the Go function.
//
// Takes an eng of type gnet.Engine.
// Returns an action of type gnet.Action.
func (s *server) OnBoot(eng gnet.Engine) (action gnet.Action) {
	log.Printf("running server on %s with multi-core=%t",
		fmt.Sprintf("%s://%s", s.network, s.addr), s.multicore)
	s.eng = eng
	return
}

// OnOpen description of the Go function.
//
// Parameters:
//
//	c gnet.Conn
//
// Return types:
//
//	out []byte, action gnet.Action
func (s *server) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	c.SetContext(new(protocol.LengthFieldBasedFrameCodec))
	atomic.AddInt32(&s.connected, 1)
	out = []byte(protocol.ConnectedAck)
	return
}

// OnTraffic handles the traffic for the server.
//
// It takes a gnet.Conn as a parameter and returns a gnet.Action.
func (s *server) OnTraffic(c gnet.Conn) (action gnet.Action) {
	codec := c.Context().(*protocol.LengthFieldBasedFrameCodec)
	data, err := codec.Decode(c)
	if err == protocol.ErrIncompletePacket {
		return
	}
	if err != nil {
		log.Fatalf("invalid packet: %v", err)
	}
	log.Printf("receive data length(%d)\n", len(data))
	dataBuffer <- data
	// resp := <-respBuffer
	// log.Println(resp)
	// res := resp.EncodeToByte()
	// _, _ = c.Write([]byte{res})
	return
}

func RunServer(addr string) {
	go received()

	ss := &server{
		network:   "tcp",
		addr:      addr,
		multicore: false,
	}
	err := gnet.Run(ss, ss.network+"://"+ss.addr, gnet.WithMulticore(false))
	log.Printf("server exits with error: %v\n", err)
}

func received() {
	for {
		data := <-dataBuffer
		msgRec := message.MinioMessage{}
		err := proto.Unmarshal(data, &msgRec)
		if err != nil {
			fmt.Println(data)
			log.Panicln(err)
		}
		err = minio.ProcessMinioEvent(&msgRec)
		if err != nil {
			log.Panicln(err)
		}
		// resp := &message.RespMessage{
		// 	Seq: msgRec.Seq,
		// 	Ok:  true,
		// }
		// respBuffer <- resp
	}
}
