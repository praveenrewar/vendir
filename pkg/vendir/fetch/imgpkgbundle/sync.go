// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0
package imgpkgbundle

import (
	"fmt"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
	ctlimg "carvel.dev/vendir/pkg/vendir/fetch/image"
	ctlver "carvel.dev/vendir/pkg/vendir/versions"
)

type Sync struct {
	opts   ctlconf.DirectoryContentsImgpkgBundle
	imgpkg *ctlimg.Imgpkg
}

func NewSync(opts ctlconf.DirectoryContentsImgpkgBundle, refFetcher ctlfetch.RefFetcher, c ctlcache.Cache) *Sync {
	imgpkgOpts := ctlimg.ImgpkgOpts{
		SecretRef:              opts.SecretRef,
		DangerousSkipTLSVerify: opts.DangerousSkipTLSVerify,
		ResponseHeaderTimeout:  opts.ResponseHeaderTimeout,
	}
	return &Sync{opts, ctlimg.NewImgpkg(imgpkgOpts, refFetcher, c)}
}

func (t Sync) Desc() string {
	image := "?"
	if len(t.opts.Image) > 0 {
		image = t.opts.Image
		if t.opts.TagSelection != nil {
			image += ":tag=" + t.opts.TagSelection.Description()
		}
	}
	return image
}

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsImgpkgBundle, error) {
	lockConf := ctlconf.LockDirectoryContentsImgpkgBundle{}

	image, err := t.resolveImage()
	if err != nil {
		return lockConf, err
	}

	var imgRef string
	if !t.opts.Recursive {
		imgRef, err = t.imgpkg.FetchBundle(image, dstPath)
		if err != nil {
			return lockConf, err
		}
	} else {
		imgRef, err = t.imgpkg.FetchBundleRecursively(image, dstPath)
		if err != nil {
			return lockConf, err
		}
	}

	lockConf.Image = imgRef
	if len(t.opts.PreresolvedTag()) > 0 {
		lockConf.Tag = t.opts.PreresolvedTag()
	} else {
		lockConf.Tag = ctlimg.NewGuessedRefParts(image).Tag
	}

	return lockConf, nil
}

func (t *Sync) resolveImage() (string, error) {
	if len(t.opts.Image) == 0 {
		return "", fmt.Errorf("Expected non-empty image")
	}

	if t.opts.TagSelection != nil {
		tags, err := t.imgpkg.Tags(t.opts.Image)
		if err != nil {
			return "", err
		}

		selectedTag, err := ctlver.HighestConstrainedVersion(tags, *t.opts.TagSelection)
		if err != nil {
			return "", fmt.Errorf("Determining tag selection: %s", err)
		}

		// In case image erroneously contains tag or digest,
		// pull operation will fail, so no need to do any checks here.
		return t.opts.Image + ":" + selectedTag, nil
	}

	return t.opts.Image, nil
}
