package image

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/legacy/tarball"
	imgName "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

var tarballSeedRef imgName.Reference

func init() {
	tarballSeedRef, _ = imgName.ParseReference("dequash-seed-tarball:v0")
}

// TODO sha256 hex will took up 64 characters, maybe use 36-base number to cut down the length
func PushSeedImage(name, seed string) (diffID string, err error) {
	img, err := RandomImageFromSeed(seed)
	if err != nil {
		return "", fmt.Errorf("generate seed image: %w", err)
	}
	lys, err := img.Layers()
	if err != nil {
		return "", fmt.Errorf("get image layers: %w", err)
	}
	id, err := lys[0].DiffID()
	if err != nil {
		return "", fmt.Errorf("get the 1st diffID of the seed image: %w", err)
	}
	diffID = id.Hex

	inp, err := parseImageName(name)
	if err != nil {
		return "", fmt.Errorf("parse image name %q: %s", name, err)
	}
	if inp.source == SourceRegistry {
		keychain := authn.DefaultKeychain
		if len(inp.username) > 0 {
			keychain = newKeychainToSingleUser(inp.username, inp.passowrd)
		}
		if err := remote.Write(inp.ref, img, remote.WithAuthFromKeychain(keychain)); err != nil {
			return "", fmt.Errorf("write image to registry: %w", err)
		}
	} else {
		// TODO test tarball seed
		tf, err := os.OpenFile(inp.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("create tarball seed %q: %w", inp.path, err)
		}
		if err := tarball.Write(tarballSeedRef, img, tf); err != nil {
			return "", fmt.Errorf("write tarball seed %q: %w", inp.path, err)
		}
	}
	return
}

func PullSeedImage(name string) (diffID string, err error) {
	inp, err := parseImageName(name)
	if err != nil {
		err = fmt.Errorf("parse image name %q: %w", name, err)
		return
	}
	var img v1.Image
	if inp.source == SourceRegistry {
		img, err = PullFromRegistry(inp.ref, inp.username, inp.passowrd)
	} else {
		img, err = PullFromTarball(inp.path)
	}
	if err != nil {
		return
	}
	lys, err := img.Layers()
	if err != nil {
		err = fmt.Errorf("get seed image layers: %w", err)
		return
	}
	if len(lys) != 1 {
		err = fmt.Errorf("expected seed image has exactly 1 layer, got %d layers", len(lys))
		return
	}
	id, err := lys[0].DiffID()
	if err != nil {
		err = fmt.Errorf("get diffID of the 1st layer of the seed image: %w", err)
		return
	}
	diffID = id.Hex
	return
}

func RandomImageFromSeed(seed string) (v1.Image, error) {
	b := &bytes.Buffer{}
	hasher := sha256.New()
	mw := io.MultiWriter(b, hasher)

	tw := tar.NewWriter(mw)
	th := &tar.Header{
		Name:     seed,
		Size:     0,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(th); err != nil {
		return nil, fmt.Errorf("write random file to layer tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close layer tar writer: %w", err)
	}
	h := v1.Hash{
		Algorithm: "sha256",
		Hex:       hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size()))),
	}
	layer, err := partial.UncompressedToLayer(&uncompressedLayer{
		diffID:    h,
		mediaType: types.DockerLayer,
		content:   b.Bytes(),
	})
	if err != nil {
		return nil, fmt.Errorf("create random layer: %w", err)
	}
	img, err := mutate.Append(empty.Image, mutate.Addendum{Layer: layer})
	if err != nil {
		return nil, fmt.Errorf("create random image: %w", err)
	}
	return img, nil
}

// uncompressedLayer implements partial.UncompressedLayer from raw bytes.
type uncompressedLayer struct {
	diffID    v1.Hash
	mediaType types.MediaType
	content   []byte
}

var _ partial.UncompressedLayer = (*uncompressedLayer)(nil)

// DiffID implements partial.UncompressedLayer
func (ul *uncompressedLayer) DiffID() (v1.Hash, error) {
	return ul.diffID, nil
}

// Uncompressed implements partial.UncompressedLayer
func (ul *uncompressedLayer) Uncompressed() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewBuffer(ul.content)), nil
}

// MediaType returns the media type of the layer
func (ul *uncompressedLayer) MediaType() (types.MediaType, error) {
	return ul.mediaType, nil
}