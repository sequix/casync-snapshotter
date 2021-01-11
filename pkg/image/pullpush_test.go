package image

//func TestParseImageName(t *testing.T) {
//	litter.Dump(parseImageName("test"))
//	litter.Dump(parseImageName("docker.io/library/nginx:1.18.0-alpine"))
//	litter.Dump(parseImageName("http://test:pass@127.0.0.1:5000/nginx:1.18-edited"))
//	litter.Dump(parseImageName("http://127.0.0.1:5000/nginx:1.18-edited"))
//	litter.Dump(parseImageName("https://test:pass@127.0.0.1:5000/nginx:1.18-edited"))
//	litter.Dump(parseImageName("registry:http://test:pass@127.0.0.1:5000/nginx:1.18-edited"))
//	litter.Dump(parseImageName("tarball:nginx.tar"))
//}

func TestPush(t *testing.T) {
	img, err := random.Image(64, 1)
	if err != nil {
		log.Fatal(err)
	}
	litter.Dump(img.Manifest())
	lys, _ := img.Layers()
	litter.Dump(lys[0].DiffID())
}