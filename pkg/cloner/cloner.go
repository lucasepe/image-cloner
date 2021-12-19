package cloner

import (
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type Cloner interface {
	GetTargetRegistry() string
	CloneEventually(ref string) (string, error)
}

type Credentials struct {
	Username string
	Password string
}

func New(reg string, creds Credentials) *imageCloner {
	login := authn.AuthConfig{
		Username: creds.Username,
		Password: creds.Password,
	}

	return &imageCloner{
		targetRegistry: reg,
		targetAuth:     authn.FromConfig(login),
	}
}

var _ Cloner = (*imageCloner)(nil)

type imageCloner struct {
	targetRegistry string
	targetAuth     authn.Authenticator
}

func (ic *imageCloner) GetTargetRegistry() string {
	return ic.targetRegistry
}

func (ic *imageCloner) CloneEventually(src string) (string, error) {
	dstRef, err := name.NewTag(src, name.WithDefaultRegistry(ic.targetRegistry))
	if err != nil {
		return "", err
	}

	clone, err := ic.shouldClone(dstRef)
	if err != nil {
		return "", err
	}
	if !clone {
		return dstRef.Name(), nil
	}

	if err := ic.copyImage(src, dstRef); err != nil {
		return "", err
	}

	return dstRef.Name(), nil
}

func (ic *imageCloner) copyImage(src string, dstRef name.Tag) error {
	srcRef, err := name.ParseReference(src)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %w", src, err)
	}

	img, err := remote.Image(srcRef)
	if err != nil {
		return fmt.Errorf("fetching %q: %w", src, err)
	}

	return remote.Write(dstRef, img, remote.WithAuth(ic.targetAuth))
}

func (ic *imageCloner) shouldClone(dstRef name.Reference) (bool, error) {
	img, err := remote.Image(dstRef, remote.WithAuth(ic.targetAuth))
	if err != nil {
		ex, ok := err.(*transport.Error)
		if ok {
			if ex.StatusCode != http.StatusNotFound {
				return false, fmt.Errorf("reading image %q: %w", dstRef, err)
			}
		} else {
			return false, err
		}
	}

	return (img == nil), nil
}
