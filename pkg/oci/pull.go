package oci

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/google/go-containerregistry/pkg/authn"
	imgName "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/sequix/casync-snapshotter/pkg/log"
)

const (
	SourceRegistry = "registry"
	SourceTarball  = "tarball"
)

type imageNameProperties struct {
	source   string
	ref      imgName.Reference
	path     string
	username string
	password string
	insecure bool
}

var imageNameReg = regexp.MustCompile(`^(registry:|tarball:)?(https?://)?(([^:]+):([^@]+)@)?(.*)$`)

func parseImageName(name string) (*imageNameProperties, error) {
	ms := imageNameReg.FindStringSubmatch(name)
	if len(ms) != 7 {
		return nil, fmt.Errorf("invalid image name %s", name)
	}
	inp := &imageNameProperties{}
	switch ms[1] {
	case "tarball:":
		inp.source = SourceTarball
		inp.path = ms[6]
	case "registry:", "":
		inp.source = SourceRegistry
		inp.insecure = ms[2] == "http://"
		inp.username = ms[4]
		inp.password = ms[5]
		var parseOpts []imgName.Option
		if inp.insecure {
			parseOpts = append(parseOpts, imgName.Insecure)
		}
		ref, err := imgName.ParseReference(ms[6], parseOpts...)
		if err != nil {
			return nil, fmt.Errorf("parse image name %s: %w", name, err)
		}
		inp.ref = ref
	default:
		return nil, fmt.Errorf("unkown image source %s, want one of [registry:, tarball:]", ms[1])
	}
	return inp, nil
}

func Pull(name string) (v1.Image, error) {
	inp, err := parseImageName(name)
	if err != nil {
		return nil, fmt.Errorf("parse image name %s: %w", name, err)
	}
	if inp.source == SourceTarball {
		return PullFromTarball(inp.path)
	}
	var parseOpts []imgName.Option
	if inp.insecure {
		parseOpts = append(parseOpts, imgName.Insecure)
	}
	return PullFromRegistry(inp.ref, inp.username, inp.password)
}

func PullFromRegistry(ref imgName.Reference, username, password string) (v1.Image, error) {
	keychain := authn.DefaultKeychain
	if len(username) > 0 {
		keychain = newKeychainToSingleUser(username, password)
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(keychain))
	if err != nil {
		return nil, fmt.Errorf("pull image %s: %w", ref.Name(), err)
	}
	return img, nil
}

func PullFromTarball(path string) (v1.Image, error) {
	tagStr, err := getTagFromTarball(path)
	if err != nil {
		return nil, err
	}
	tag, err := imgName.NewTag(tagStr)
	if err != nil {
		return nil, fmt.Errorf("parse tag %s for tarball %s: %w", tagStr, path, err)
	}
	img, err := tarball.ImageFromPath(path, &tag)
	if err != nil {
		return nil, fmt.Errorf("pull image from taball %s with tag %s: %w", path, tagStr, err)
	}
	return img, err
}

// only return the first tag
func getTagFromTarball(path string) (string, error) {
	tarfile, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file %s: %w", path, err)
	}
	defer func() {
		if cerr := tarfile.Close(); cerr != nil {
			log.G.WithError(cerr).With("path", path).Warn("close file")
		}
	}()
	var tag string
	tr := tar.NewReader(tarfile)
	for {
		th, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf(`not found "manifest.json" in %s`, path)
		}
		if err != nil {
			return "", fmt.Errorf("read tar file %s: %w", path, err)
		}
		if th.Name == "manifest.json" {
			manifestContent, err := ioutil.ReadAll(tr)
			if err != nil {
				return "", fmt.Errorf(`read "manifest.json" from %s: %w"`, path, err)
			}
			manifest := []map[string]interface{}{}
			if err := json.Unmarshal(manifestContent, &manifest); err != nil {
				return "", fmt.Errorf(`unmarshal "manifest.json" from %s: %w`, path, err)
			}
			if len(manifest) != 1 {
				return "", fmt.Errorf(`got %d images from "manifest.json" from %s, want exactly 1`,
					len(manifest), path)
			}
			repoTagsI, ok := manifest[0]["RepoTags"]
			if !ok || repoTagsI == nil {
				return "", fmt.Errorf(`got no "RepoTags" in "manifest.json" from %s`, path)
			}
			repoTags, ok := repoTagsI.([]interface{})
			if !ok || len(repoTags) != 1 {
				return "", fmt.Errorf(`got %d tags in "manifest.json" from %s, want exactly 1`,
					len(repoTags), path)
			}
			tag, ok = repoTags[0].(string)
			if !ok {
				return "", fmt.Errorf(`got the only tag as %t from %s, want string`, repoTags[0], path)
			}
			break
		}
	}
	if len(tag) == 0 {
		return "", fmt.Errorf("not found any tag in %s", path)
	}
	return tag, nil
}

type keychainToSingleUser struct {
	auth authn.Authenticator
}

// Resolve looks up the most appropriate credential for the specified target.
func (k *keychainToSingleUser) Resolve(_ authn.Resource) (authn.Authenticator, error) {
	return k.auth, nil
}

func newKeychainToSingleUser(username, password string) authn.Keychain {
	auth := authn.FromConfig(authn.AuthConfig{
		Username: username,
		Password: password,
	})
	return &keychainToSingleUser{auth}
}
