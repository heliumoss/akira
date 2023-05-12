package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
)

func processImage(byteContainer []byte, size string, quality int) (Image, error) {
	sizeSplit := strings.Split(size, "x")
	width, err := strconv.Atoi(sizeSplit[0])
	if err != nil {
		return Image{}, err
	}
	height, err := strconv.Atoi(sizeSplit[1])
	if err != nil {
		return Image{}, err
	}

	image := bimg.NewImage(byteContainer)

	imageSize, err := image.Size()
	if err != nil {
		return Image{}, err
	}
	_, err = image.Convert(bimg.WEBP)
	if err != nil {
		return Image{}, err
	}

	if imageSize.Width >= width || imageSize.Height >= height {
		_, err = image.Resize(width, height)
		if err != nil {
			return Image{}, err
		}
	} else {
		_, err = image.Enlarge(width, height)
		if err != nil {
			return Image{}, err
		}
	}

	compression, err := image.Process(bimg.Options{
		Quality: quality,
	})
	if err != nil {
		return Image{}, err
	}

	encoded := fmt.Sprintf("data:image/webp;base64,%s", base64.StdEncoding.EncodeToString(compression))
	return Image{Size: size, Base64: encoded}, nil
}
