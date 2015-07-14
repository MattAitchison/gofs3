package main

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rjeczalik/notify"
)

var src, dst string

func main() {
	done := make(chan bool)
	if len(os.Args) != 3 {
		log.Fatal("src and dst required. gofs3 <src> <dst>")
	}
	src, dst = os.Args[1], os.Args[2]
	var err error
	src, err = filepath.Abs(src)
	if err != nil {
		log.Fatal(err)
	}

	svc := s3.New(&aws.Config{
		Endpoint: "s3.amazonaws.com",
	})

	uploader := s3manager.NewUploader(&s3manager.UploadOptions{S3: svc})

	// Make the channel buffered to ensure no event is dropped. Notify will drop
	// an event if the receiver is not able to keep up the sending pace.
	c := make(chan notify.EventInfo, 1)

	// Set up a watchpoint listening on events within current working directory.
	// Dispatch each create and remove events separately to c.
	if err := notify.Watch(filepath.Join(src, "..."), c, notify.Create, notify.Remove, notify.Write); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)
	log.Println("watching...")

	// Block until an event is received.
	for e := range c {
		log.Println(e)
		switch e.Event() {
		case notify.Create:
			// handleCreate(uploader, e.Path())
		case notify.Write:
			handleCreate(uploader, e.Path())
		case notify.Remove:
			handleRemove(svc, dst, e.Path())
		}
	}
	// ei := <-c

	<-done

}

// type S3Sync struct {
// 	Bucket string
// 	*s3.S3
// }

func handleRemove(svc *s3.S3, bucket, path string) error {
	key := strings.TrimPrefix(path, src)
	res, err := svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})

	if err != nil {
		log.Fatal(err)
	}

	log.Println(res)
	return nil
}

func handleCreate(u *s3manager.Uploader, path string) error {

	r, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return err
	}

	key := strings.TrimPrefix(path, src)
	res, err := u.Upload(&s3manager.UploadInput{
		Bucket: aws.String(dst),
		Body:   r,
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println(res)
	return nil
}
