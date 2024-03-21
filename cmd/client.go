package cmd

import (
	"bufio"
	"log"
	"net"

	"google.golang.org/protobuf/proto"

	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/robfig/cron/v3"
	"github.com/yimiaoxiehou/minio-sync/internal/message"
	"github.com/yimiaoxiehou/minio-sync/internal/minio"
	"github.com/yimiaoxiehou/minio-sync/internal/protocol"
)

func RunClient(addr string, skipBuckets []string, appendOnly bool) {
	c, err := net.Dial("tcp", addr)
	logErr(err)
	logging.Infof("connection=%s starts...", c.LocalAddr().String())
	defer func() {
		logging.Infof("connection=%s stops...", c.LocalAddr().String())
		c.Close()
	}()
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	logErr(err)
	if string(msg) != protocol.ConnectedAck {
		logging.Fatalf("the first response packet mismatches, expect: \"%s\", but got: \"%s\"", protocol.ConnectedAck, msg)
	}
	go sendAndRecv(c, rd)

	minio.ExportAllObject(reqBuffer)
	// 同步iam信息
	reqBuffer <- minio.ExportIAM()
	if !appendOnly {
		// 同步bucket信息
		for _, m := range minio.ExpoortBucketMetadata(skipBuckets) {
			reqBuffer <- m
		}
	}

	// Schedule a cron job to export IAM and Minio buckets data every 5 minutes
	cr := cron.New()
	_, err = cr.AddFunc("@every 2h", func() {
		// 同步iam信息
		reqBuffer <- minio.ExportIAM()
		// 同步bucket信息

		if !appendOnly {
			// 同步bucket信息
			for _, m := range minio.ExpoortBucketMetadata(skipBuckets) {
				reqBuffer <- m
			}
		}
	})
	logErr(err)
	cr.Start()
	minio.ListenMinioBucketEvent(skipBuckets, reqBuffer)
}

func logErr(err error) {
	logging.Error(err)
	if err != nil {
		panic(err)
	}
}

var reqBuffer chan (*message.MinioMessage) = make(chan (*message.MinioMessage), 8)

func sendAndRecv(c net.Conn, rd *bufio.Reader) {
	for {
		msg := <-reqBuffer
		log.Printf("send message seq(%d) type(%s) bucket(%s) name(%s)\n", msg.GetSeq(), msg.GetType().String(), msg.GetBucket(), msg.GetName())
		codec := protocol.LengthFieldBasedFrameCodec{}
		encodingMsg, err := proto.Marshal(msg)
		logErr(err)
		packet, err := codec.Encode(encodingMsg)
		logErr(err)
		_, err = c.Write(packet)
		logErr(err)
		bs, err := rd.ReadByte()
		logErr(err)
		resp := message.DecodeFromByte(bs)
		log.Printf("receive resp seq(%d) ok(%t)\n", resp.GetSeq(), resp.GetOk())
	}
}
