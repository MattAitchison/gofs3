package main

import (
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"gopkg.in/fsnotify.v1"
)

var src, dst string

func main() {
	if len(os.Args) != 3 {
		log.Fatal("wrong number of arguments")
	}
	src, dst = os.Args[1], os.Args[2]

	// creds := credentials.NewEnvCredentials()
	// if _, err := creds.Get(); err != nil {
	// 	log.Fatal(err)
	// }
	//
	// svc := s3.New(&aws.Config{
	// 	Credentials:      creds,
	// 	Region:           "us-east-2",
	// 	Endpoint:         "s3.amazonaws.com",
	// 	S3ForcePathStyle: true,
	// })
	svc := s3.New(&aws.Config{
		Endpoint: "s3.amazonaws.com",
		// S3ForcePathStyle: true,
	})

	uploader := s3manager.NewUploader(&s3manager.UploadOptions{S3: svc})
	// uploader := s3manager.NewUploader(nil)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				switch event.Op {
				case fsnotify.Create:
					handleCreate(watcher, uploader, event.Name)
				case fsnotify.Write:
					log.Println("modified file:", event.Name)
				}

			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			log.Println(path)
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	<-done

}

func handleCreate(watcher *fsnotify.Watcher, u *s3manager.Uploader, path string) error {
	fi, _ := os.Lstat(path)

	if fi.IsDir() {
		return watcher.Add(path)
	}

	r, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return err
	}

	name := fi.Name()
	_, err = u.Upload(&s3manager.UploadInput{
		Bucket: aws.String(dst),
		Body:   r,
		Key:    aws.String(name),
	})
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
