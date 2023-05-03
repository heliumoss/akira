package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
)

func processImage(byteContainer []byte, size string) (Image, error) {
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

	_, err = image.Convert(bimg.JPEG)
	if err != nil {
		return Image{}, err
	}

	resized, err := image.Resize(width, height)
	if err != nil {
		return Image{}, err
	}

	encoded := fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(resized))

	return Image{Size: size, Base64: encoded}, nil
}
