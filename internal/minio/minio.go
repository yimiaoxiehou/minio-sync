package minio

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	idgenerator "github.com/yimiaoxiehou/minio-sync/internal/id_generator"
	"github.com/yimiaoxiehou/minio-sync/internal/message"
)

var aClient *madmin.AdminClient
var mClient *minio.Client

func InitMinioClient(minioAddress, minioUsername, minioPassword string) {
	log.Printf("Connect minio address(%s) username(%s)\n", minioAddress, minioUsername)
	var err error
	// Initialize MinIO admin client
	aClient, err = madmin.New(minioAddress, minioUsername, minioPassword, false)
	logErr(err)

	// Initialize MinIO client
	mClient, err = minio.New(minioAddress, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUsername, minioPassword, ""),
		Secure: false,
	})
	logErr(err)
	_, err = mClient.HealthCheck(time.Second * 3)
	logErr(err)
	log.Printf("Connected minio address(%s) username(%s)\n", minioAddress, minioUsername)
}

func ProcessMinioEvent(msg *message.MinioMessage) error {
	log.Printf("process msg seq(%d) type(%s) bucket(%s) name(%s)\n", msg.GetSeq(), msg.GetType().String(), msg.GetBucket(), msg.GetName())
	defer log.Printf("process msg seq(%d) type(%s) bucket(%s) name(%s) done\n", msg.GetSeq(), msg.GetType().String(), msg.GetBucket(), msg.GetName())
	switch msg.GetType().Number() {
	case message.MessageType_Minio_IAM_Export.Number():
		return aClient.ImportIAM(context.Background(), io.NopCloser(bytes.NewBuffer(msg.GetContent())))
	case message.MessageType_Minio_BUCKETS_Export.Number():
		_, err := aClient.ImportBucketMetadata(context.Background(), msg.GetBucket(), io.NopCloser(bytes.NewBuffer(msg.GetContent())))
		return err
	case message.MessageType_S3_Object_Put.Number():
		obj, err := mClient.StatObject(context.Background(), msg.GetBucket(), msg.GetName(), minio.StatObjectOptions{})
		// obj exist and accessable
		if err == nil && obj.ETag == msg.GetEtag() {
			return nil
		}
		read := bytes.NewReader(msg.GetContent())
		_, err = mClient.PutObject(context.Background(), msg.GetBucket(), msg.GetName(), read, int64(len(msg.GetContent())), minio.PutObjectOptions{})
		return err
	case message.MessageType_S3_Obejct_Delete.Number():
		return mClient.RemoveObject(context.Background(), msg.GetBucket(), msg.GetName(), minio.RemoveObjectOptions{})
	}
	return nil
}

func ListenMinioBucketEvent(skipBuckets []string, reqBuffer chan *message.MinioMessage) {
	// Listen for Minio bucket notifications and handle events
	for notificationInfo := range mClient.ListenNotification(context.Background(), "", "", []string{
		"s3:ObjectCreated:Put",
		"s3:ObjectRemoved:Delete",
	}) {
		if notificationInfo.Err != nil {
			log.Fatalln(notificationInfo.Err)
		}
		if slices.Contains(skipBuckets, notificationInfo.Records[0].S3.Bucket.Name) {
			continue
		}
		for _, record := range notificationInfo.Records {
			if "s3:ObjectRemoved:Delete" == record.EventName {
				msg := message.MinioMessage{
					Seq:     idgenerator.GetInstance().Get(),
					Type:    message.MessageType_S3_Obejct_Delete,
					Bucket:  record.S3.Bucket.Name,
					Name:    record.S3.Object.Key,
					Etag:    record.S3.Object.ETag,
					Content: nil,
				}
				reqBuffer <- &msg
			}
			if "s3:ObjectCreated:Put" == record.EventName {
				obj, err := mClient.GetObject(context.Background(), record.S3.Bucket.Name, record.S3.Object.Key, minio.GetObjectOptions{})
				logErr(err)
				cont, err := io.ReadAll(obj)
				logErr(err)
				msg := message.MinioMessage{
					Seq:     idgenerator.GetInstance().Get(),
					Type:    message.MessageType_S3_Object_Put,
					Bucket:  record.S3.Bucket.Name,
					Name:    record.S3.Object.Key,
					Etag:    record.S3.Object.ETag,
					Content: cont,
				}
				reqBuffer <- &msg
			}
		}
	}
}

func ExpoortBucketMetadata(skipBuckets []string) []*message.MinioMessage {
	buckets, err := mClient.ListBuckets(context.Background())
	logErr(err)
	var msgs []*message.MinioMessage
	for _, bucket := range buckets {
		if slices.Contains(skipBuckets, bucket.Name) {
			continue
		}
		reader, err := aClient.ExportBucketMetadata(context.Background(), bucket.Name)
		logErr(err)
		data, err := io.ReadAll(reader)
		logErr(err)
		msgs = append(msgs, &message.MinioMessage{
			Seq:     idgenerator.GetInstance().Get(),
			Type:    message.MessageType_Minio_BUCKETS_Export,
			Bucket:  bucket.Name,
			Name:    "",
			Etag:    "",
			Content: data,
		})
	}
	return msgs
}

func ExportIAM() *message.MinioMessage {
	// Export IAM settings and capture the response
	r, err := aClient.ExportIAM(context.Background())
	logErr(err)

	data, err := io.ReadAll(r)
	logErr(err)

	return &message.MinioMessage{
		Type:    message.MessageType_Minio_IAM_Export,
		Bucket:  "",
		Name:    "",
		Etag:    "",
		Content: data,
	}
}

func ExportAllObject(reqBuffer chan *message.MinioMessage) {
	log.Println("export all object.")
	bks, err := mClient.ListBuckets(context.Background())
	logErr(err)
	for _, bk := range bks {
		listBucketAllObj(bk.Name, "", reqBuffer)
	}
}

func listBucketAllObj(bk, prefix string, reqBuffer chan *message.MinioMessage) {
	for obj := range mClient.ListObjects(context.Background(), bk, minio.ListObjectsOptions{Prefix: prefix}) {
		if obj.Size == 0 && strings.HasSuffix(obj.Key, string(os.PathSeparator)) {
			listBucketAllObj(bk, obj.Key, reqBuffer)
			continue
		}
		r, err := mClient.GetObject(context.Background(), bk, obj.Key, minio.GetObjectOptions{})
		logErr(err)
		cont, err := io.ReadAll(r)
		if err != nil {
			logErr(err)
		}
		msg := message.MinioMessage{
			Seq:     idgenerator.GetInstance().Get(),
			Type:    message.MessageType_S3_Object_Put,
			Bucket:  bk,
			Name:    obj.Key,
			Etag:    obj.ETag,
			Content: cont,
		}
		reqBuffer <- &msg
	}
}

func logErr(err error) {
	logging.Error(err)
	if err != nil {
		panic(err)
	}
}
