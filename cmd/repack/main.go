package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/chanim"
)

type options struct {
	InputDir  string `short:"i" long:"input-dir"  description:"The input directory"`
	OutputDir string `short:"o" long:"output-dir" description:"The output directory"`
	NotRotate bool   `short:"n" long:"not-rotate" description:"Disable image rotate"`
}

func images(opts options) chan string {
	ch := make(chan string, 512)
	go func() {
		defer close(ch)

		walkFn := func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				if isImage, _ := filepath.Match("*.png", info.Name()); isImage {
					ch <- path
				}
			}
			return err
		}

		err := filepath.Walk(opts.InputDir, walkFn)
		if err != nil {
			panic(err)
		}
	}()
	return ch
}

func parseCmd() options {
	var opts options
	var cmdParser = flags.NewParser(&opts, flags.Default)
	var err error

	if _, err = cmdParser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			panic(err)
		}
	}

	if opts.InputDir, err = filepath.Abs(opts.InputDir); err != nil {
		panic(err)
	}

	if opts.OutputDir, err = filepath.Abs(opts.OutputDir); err != nil {
		panic(err)
	}

	return opts
}

func pixmapSize(pixmap *chanim.Pixmap) int64 {
	return int64(pixmap.BytePerLine * pixmap.Height)
}

func eqPixels(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func rotatePixmap(pixmap *chanim.Pixmap) *chanim.Pixmap {
	pixSize := chanim.GetPixelSize(pixmap.PixFormat)
	rotatedData := make([]byte, 0, pixmap.Width*pixmap.Height*pixSize)
	for x := 0; x < pixmap.Width; x++ {
		for y := pixmap.Height - 1; y >= 0; y-- {
			pixOffset := y*pixmap.BytePerLine + x*pixSize
			rotatedData = append(rotatedData, pixmap.Data[pixOffset:pixOffset+pixSize]...)
		}
	}

	return &chanim.Pixmap{
		Data:        rotatedData,
		Width:       pixmap.Height,
		Height:      pixmap.Width,
		PixFormat:   pixmap.PixFormat,
		BytePerLine: pixSize * pixmap.Height,
	}
}

func savePackedPixmap(opts *options, inputImageFile string, packedPixmap *chanim.PackedPixmap) {
	relInputPath, err := filepath.Rel(opts.InputDir, inputImageFile)
	if err != nil {
		panic(err)
	}
	relImageDir := filepath.Dir(relInputPath)

	outputImageDir := filepath.Join(opts.OutputDir, relImageDir)
	err = os.MkdirAll(outputImageDir, 0755)
	if err != nil {
		panic(err)
	}

	inputImageExt := filepath.Ext(inputImageFile)
	relOutputPath := strings.TrimSuffix(relInputPath, inputImageExt) + ".ppixmap"
	outputFile := filepath.Join(opts.OutputDir, relOutputPath)
	err = packedPixmap.Save(outputFile)
	if err != nil {
		panic(err)
	}
}

func removeOutputDir(opts *options) {
	if err := os.RemoveAll(opts.OutputDir); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}
}

func main() {
	opts := parseCmd()

	removeOutputDir(&opts)

	var packedSize int64
	var unpackedSize int64
	for imageFile := range images(opts) {
		fmt.Printf("Processing %s\n", imageFile)

		pixmap, err := chanim.LoadPixmap(imageFile, chanim.RGB16)
		if err != nil {
			panic(err)
		}
		unpackedSize += pixmapSize(pixmap)
		if !opts.NotRotate {
			pixmap = rotatePixmap(pixmap)
		}

		packedPixmap, err := chanim.PackPixmap(pixmap)
		if err != nil {
			panic(err)
		}
		packedSize += int64(len(packedPixmap.Data))

		savePackedPixmap(&opts, imageFile, packedPixmap)
	}

	fmt.Printf("---------------------------\n")
	fmt.Printf("unpackedSize=%vM\n", float32(unpackedSize)/float32(1024*1024))
	fmt.Printf("packedSize=%vM\n", float32(packedSize)/float32(1024*1024))
	fmt.Printf("unpackedSize/packedSize=%v\n", float32(unpackedSize)/float32(packedSize))
}
