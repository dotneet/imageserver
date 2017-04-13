// Package pngquant provides a imageserver/image.Processor implementation that allows to lossy compress for png.
package pngquant

import (
	"image"

	"github.com/pierrre/imageserver"
	"os/exec"
	"strings"
	"bytes"
)

// Processor is a imageserver/image.Processor implementation that allows to pngquant Image.

// pngquant
type Processor struct{
	Command string
	Speed 	string
	EnableMaxArea int
}

// Process implements imageserver/image.Processor.
func (prc *Processor) Process(im *imageserver.Image, params imageserver.Params) (*imageserver.Image, error) {
	data, err := prc.compress(im.Data)
	if err != nil {
		return nil, err
	}
	return  &imageserver.Image{Format:im.Format, Data: data}, nil
}

// Change implements imageserver/image.Processor.
func (prc *Processor) Change(im *imageserver.Image, params imageserver.Params) bool {
	config, _, err := image.DecodeConfig(bytes.NewReader(im.Data))
	if err != nil {
		return true
	}
	if prc.EnableMaxArea > 0 && config.Width * config.Height > prc.EnableMaxArea {
		return  false
	}
	return  true
}

func (prc *Processor) compress(input []byte) (output []byte, err error) {
	speed := prc.Speed
	if speed == "" {
		speed = "3"
	}

	cmd := exec.Command(prc.Command, "-", "--speed", speed)
	cmd.Stdin = strings.NewReader(string(input))
	var o bytes.Buffer
	cmd.Stdout = &o
	err = cmd.Run()
	if err != nil {
		return
	}

	output = o.Bytes()
	return
}
