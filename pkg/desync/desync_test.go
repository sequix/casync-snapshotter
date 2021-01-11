package desync

import (
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/sanity-io/litter"
)

func Test(t *testing.T) {
	litter.Dump(humanize.ParseBytes("1Gi"))
}
//
//func TestMinioS3Client(t *testing.T) {
//	endpoint := "s3.ap-northeast-1.amazonaws.com"
//	accessKeyID := "AKIAIM42KJKRKYGKXEBQ"
//	secretAccessKey := "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"
//	useSSL := true
//
//	// Initialize minio client object.
//	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
//	if err != nil {
//		log.Fatalln(err)
//	}
//
//	log.Printf("%#v\n", minioClient) // minioClient is now setup
//
//	buckets, err := minioClient.ListBuckets()
//	if err != nil {
//		log.Fatalf("listing buckets: %s", err)
//	}
//
//	litter.Dump(buckets)
//}
//
//// 创建s3 store需要：bucket name、region、access key、secret key(可以从env拿，也可以程序中设置)
//func TestDesyncS3Store(t *testing.T) {
//	endpoint := "s3+https://s3.amazonaws.com/sequix/chunks"
//	accessKeyID := "AKIAIM42KJKRKYGKXEBQ"
//	secretAccessKey := "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"
//
//	//if err := os.Setenv("AWS_ACCESS_KEY", accessKeyID); err != nil {
//	//	log.Fatalf("set env AWS_ACCESS_KEY: %s", err)
//	//}
//	//
//	//if err := os.Setenv("AWS_SECRET_KEY", secretAccessKey); err != nil {
//	//	log.Fatalf("set env AWS_SECRET_KEY: %s", err)
//	//}
//	//cred := credentials.NewEnvAWS()
//
//	cred := credentials.NewStatic(accessKeyID, secretAccessKey, "", credentials.SignatureV4)
//
//	endpointURL, err := url.Parse(endpoint)
//	if err != nil {
//		log.Fatalf("parsing s3 endpoint url: %s", err)
//	}
//
//	log.Print("creating s3 store...")
//	store, err := desync.NewS3Store(endpointURL, cred, "ap-northeast-1", desync.StoreOptions{}, minio.BucketLookupAuto)
//	if err != nil {
//		log.Fatalf("creating s3 store: %s", err)
//	}
//
//	chk := desync.NewChunkFromUncompressed([]byte("the quick brown fox jumps over a lazy dog."))
//
//	log.Print("storing s3 chunk...")
//
//	// 同一个object可以被多次store
//	if err := store.StoreChunk(chk); err != nil {
//		log.Fatalf("storing chunk to s3: %s", err)
//	}
//
//	if has, err := store.HasChunk(chk.ID()); true {
//		log.Printf("has chunk %v, err %s", has, err)
//	}
//
//	//// remove时会把上层文件夹删掉，但整个文件夹为空时
//	//if err := store.RemoveChunk(chk.ID()); true {
//	//	log.Printf("remove chk: err %s", err)
//	//}
//	//
//	//if has, err := store.HasChunk(chk.ID()); true {
//	//	log.Printf("has chunk %v, err %s", has, err)
//	//}
//
//	log.Print("done")
//}
//
//// index store 不支持删除
//func TestDesyncS3IndexStore(t *testing.T) {
//	endpoint := "s3+https://s3.amazonaws.com/sequix/index"
//	accessKeyID := "AKIAIM42KJKRKYGKXEBQ"
//	secretAccessKey := "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"
//
//	//if err := os.Setenv("AWS_ACCESS_KEY", accessKeyID); err != nil {
//	//	log.Fatalf("set env AWS_ACCESS_KEY: %s", err)
//	//}
//	//
//	//if err := os.Setenv("AWS_SECRET_KEY", secretAccessKey); err != nil {
//	//	log.Fatalf("set env AWS_SECRET_KEY: %s", err)
//	//}
//	//cred := credentials.NewEnvAWS()
//
//	cred := credentials.NewStatic(accessKeyID, secretAccessKey, "", credentials.SignatureV4)
//
//	endpointURL, err := url.Parse(endpoint)
//	if err != nil {
//		log.Fatalf("parsing s3 endpoint url: %s", err)
//	}
//
//	log.Print("creating s3 index store...")
//	store, err := desync.NewS3IndexStore(endpointURL, cred, "ap-northeast-1", desync.StoreOptions{}, minio.BucketLookupAuto)
//	if err != nil {
//		log.Fatalf("creating s3 store: %s", err)
//	}
//
//	idx := desync.Index{
//		Index:  desync.FormatIndex{
//			FormatHeader: desync.FormatHeader{
//				Size: 0,
//				Type: 0,
//			},
//			FeatureFlags: 0,
//			ChunkSizeMin: 0,
//			ChunkSizeAvg: 0,
//			ChunkSizeMax: 0,
//		},
//		Chunks: nil,
//	}
//	if err := store.StoreIndex("test.caibx", idx); true {
//		log.Printf("store index err %s", err)
//	}
//
//	if has, err := store.GetIndex("test.caibx"); true {
//		log.Printf("get index %#v, err %s", has, err)
//	}
//	log.Print("done")
//}