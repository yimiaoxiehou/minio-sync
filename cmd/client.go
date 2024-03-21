package cmd

import (
	"log"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/robfig/cron/v3"
	"github.com/yimiaoxiehou/minio-sync/internal/message"
	"github.com/yimiaoxiehou/minio-sync/internal/minio"
	"github.com/yimiaoxiehou/minio-sync/internal/protocol"
	rconn "github.com/yimiaoxiehou/minio-sync/internal/reconnectconn"
)

func RunClient(addr string, skipBuckets []string, appendOnly bool) {

	c := rconn.New(addr, time.Second*3, 3, time.Second*10, func(err error) {
		log.Fatalln(err)
	})
	defer c.Close()
	go sendAndRecv(c)

	log.Println("同步 IAM 信息")
	reqBuffer <- minio.ExportIAM()
	log.Println("同步 bucket 信息")
	for _, m := range minio.ExpoortBucketMetadata(skipBuckets) {
		log.Printf("同步 bucket(%s) 信息\n", m.Name)
		reqBuffer <- m
	}
	if !appendOnly {
		minio.ExportAllObject(reqBuffer)
	}

	// Schedule a cron job to export IAM and Minio buckets data every 5 minutes
	cr := cron.New()
	_, err := cr.AddFunc("@every 2h", func() {
		log.Println("同步 IAM 信息")
		reqBuffer <- minio.ExportIAM()
		log.Println("同步 bucket 信息")
		for _, m := range minio.ExpoortBucketMetadata(skipBuckets) {
			log.Printf("同步 bucket(%s) 信息\n", m.Name)
			reqBuffer <- m
		}
	})
	logErr(err)
	cr.Start()
	minio.ListenMinioBucketEvent(skipBuckets, reqBuffer)
}

func logErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

var reqBuffer chan (*message.MinioMessage) = make(chan (*message.MinioMessage), 8)

func sendAndRecv(c *rconn.Conn) {
	for {
		msg := <-reqBuffer
		log.Printf("send message seq(%d) type(%s) bucket(%s) name(%s)\n", msg.GetSeq(), msg.GetType().String(), msg.GetBucket(), msg.GetName())
		codec := protocol.LengthFieldBasedFrameCodec{}
		encodingMsg, err := proto.Marshal(msg)
		logErr(err)
		log.Printf("send data length(%d)\n", len(encodingMsg))
		packet, err := codec.Encode(encodingMsg)
		logErr(err)
		_, err = c.Write(packet)
		logErr(err)
		log.Printf("send message seq(%d) type(%s) bucket(%s) name(%s) done\n", msg.GetSeq(), msg.GetType().String(), msg.GetBucket(), msg.GetName())
	}
}
