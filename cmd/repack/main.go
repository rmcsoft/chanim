package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/chanim"
)

type options struct {
	InputDir  string `short:"i" long:"input-dir"  required:"true" description:"The input directory"`
	OutputDir string `short:"o" long:"output-dir" required:"true" description:"The output directory"`

	NotRotate      bool `short:"n" long:"not-rotate"       description:"Disable image rotate"`
	ClearOutputDir bool `short:"c" long:"clear-output-dir" description:"Clears the output directory."`
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
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
			fail(err)
		}
	}()
	return ch
}

func parseCmd() options {
	var opts options
	var cmdParser = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var err error

	if _, err = cmdParser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			fmt.Println(flagsErr)
			os.Exit(0)
		}

		fail(err)
	}

	if opts.InputDir, err = filepath.Abs(opts.InputDir); err != nil {
		fail(err)
	}

	if opts.OutputDir, err = filepath.Abs(opts.OutputDir); err != nil {
		fail(err)
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
		fail(err)
	}
	relImageDir := filepath.Dir(relInputPath)

	outputImageDir := filepath.Join(opts.OutputDir, relImageDir)
	err = os.MkdirAll(outputImageDir, 0755)
	if err != nil {
		fail(err)
	}

	inputImageExt := filepath.Ext(inputImageFile)
	relOutputPath := strings.TrimSuffix(relInputPath, inputImageExt) + ".ppixmap"
	outputFile := filepath.Join(opts.OutputDir, relOutputPath)
	err = packedPixmap.Save(outputFile)
	if err != nil {
		fail(err)
	}
}

func clearDir(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			fail(err)
		}
		return
	}

	for _, file := range files {
		err = os.RemoveAll(path.Join(dir, file.Name()))
		if err != nil {
			fail(err)
		}
	}
}

func main() {
	opts := parseCmd()

	if opts.ClearOutputDir {
		clearDir(opts.OutputDir)
	}

	var imageCount int
	var packedSize int64
	var unpackedSize int64
	for imageFile := range images(opts) {
		fmt.Printf("Processing %s\n", imageFile)

		pixmap, err := chanim.LoadPixmap(imageFile, chanim.RGB16)
		if err != nil {
			fail(err)
		}
		unpackedSize += pixmapSize(pixmap)
		if !opts.NotRotate {
			pixmap = rotatePixmap(pixmap)
		}

		packedPixmap, err := chanim.PackPixmap(pixmap)
		if err != nil {
			fail(err)
		}
		packedSize += int64(len(packedPixmap.Data))

		savePackedPixmap(&opts, imageFile, packedPixmap)
		imageCount++
	}

	fmt.Printf("---------------------------\n")
	fmt.Printf("Processed files=%v\n", imageCount)
	fmt.Printf("unpackedSize=%vM\n", float32(unpackedSize)/float32(1024*1024))
	fmt.Printf("packedSize=%vM\n", float32(packedSize)/float32(1024*1024))
	fmt.Printf("unpackedSize/packedSize=%v\n", float32(unpackedSize)/float32(packedSize))
}
