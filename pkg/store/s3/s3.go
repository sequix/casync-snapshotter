package s3

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/store"
)

var (
	//flagS3AccessKey           = flag.String("s3-ak", "", "s3 access key, you can also specify with AWS_ACCESS_KEY_ID")
	//flagS3SecretKey           = flag.String("s3-sk", "", "s3 access secret key, you can also specify with AWS_SECRET_ACCESS_KEY")
	//flagS3Region              = flag.String("s3-region", "", "s3 region")
	//flagS3Bucket              = flag.String("s3-bucket", "", "s3 bucket")
	flagS3AccessKey           = flag.String("s3-ak", "AKIASNLQNZBTIUWCL56K", "s3 access key, you can also specify with AWS_ACCESS_KEY_ID")
	flagS3SecretKey           = flag.String("s3-sk", "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", "s3 access secret key, you can also specify with AWS_SECRET_ACCESS_KEY")
	flagS3Region              = flag.String("s3-region", "ap-northeast-1", "s3 region")
	flagS3Bucket              = flag.String("s3-bucket", "sequix", "s3 bucket")
	flagS3Prefix              = flag.String("s3-prefix", "", "prefix before all object this program will process")
	flagS3DownloadConcurrency = flag.Int("s3-download-concurrency", 8, "s3 download concurrency")
	flagS3UploadConcurrency   = flag.Int("s3-upload-concurrency", 8, "s3 upload concurrency")
)

type S3Store struct {
	session    *session.Session
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
}

func New() (*S3Store, error) {
	ak := *flagS3AccessKey
	sk := *flagS3SecretKey
	if len(ak) == 0 {
		ak = os.Getenv("AWS_ACCESS_KEY_ID")
		sk = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(*flagS3Region),
		Credentials: credentials.NewStaticCredentials(ak, sk, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 session: %w", err)
	}

	store := &S3Store{
		session: sess,
		client:  s3.New(sess),
		uploader: s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
			u.PartSize = 16 * 1024 * 1024
			u.Concurrency = *flagS3UploadConcurrency
		}),
		downloader: s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
			d.PartSize = 16 * 1024 * 1024
			d.Concurrency = *flagS3DownloadConcurrency
		}),
		bucket: *flagS3Bucket,
	}
	return store, nil
}

// TODO: no uploading for same chunk
func (s *S3Store) AddChunk(key string, src []byte) error {
	key = *flagS3Prefix + key

	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Key:    &key,
		Body:   bytes.NewReader(src),
		Bucket: &s.bucket,
	})
	if err != nil {
		return fmt.Errorf("upload s3 chunk %s: %w", key, err)
	}
	return nil
}

func (s *S3Store) GetChunk(key string, dst []byte) ([]byte, error) {
	key = *flagS3Prefix + key
	buf := aws.NewWriteAtBuffer(dst)
	read, err := s.downloader.Download(buf, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if strings.Contains(err.Error(), s3.ErrCodeNoSuchKey) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("download s3 chunk %s: read %d byte, err %w", key, read, err)
	}
	return buf.Bytes(), nil
}

func (s *S3Store) DelChunk(key string) error {
	key = *flagS3Prefix + key
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return fmt.Errorf("delete s3 chunk %s: %w", key, err)
}

func (s *S3Store) HasChunk(key string) bool {
	key = *flagS3Prefix + key

	_, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket:                     &s.bucket,
		Key:                        &key,
	})
	if err != nil {
		if strings.Contains(err.Error(), s3.ErrCodeNoSuchKey) {
			return false
		}
		log.G.WithError(err).Error("s3.HasChunk %s", key)
		return false
	}
	return true
}