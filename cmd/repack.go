package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/rmcsoft/chanim"
)

type options struct {
	InputDir  string `short:"i" long:"input-dir"  description:"The input directory"`
	OutputDir string `short:"o" long:"output-dir" description:"The output directory"`
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

	if _, err := cmdParser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
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

func savePackedPixmap(path string, packedPixmap []byte) {
	err := ioutil.WriteFile(path, packedPixmap, 0644)
	if err != nil {
		panic(err)
	}
}

func main() {
	opts := parseCmd()
	var packedSize int64
	var unpackedSize int64
	for imageFile := range images(opts) {
		fmt.Printf("Image %s\n", imageFile)

		pixmap, err := chanim.LoadPixmap(imageFile, chanim.RGB16)
		if err != nil {
			panic(err)
		}
		unpackedSize += pixmapSize(pixmap)

		packedPixmap, err := chanim.PackPixmap(pixmap)
		if err != nil {
			panic(err)
		}
		packedSize += int64(len(packedPixmap.Data))

		outputFile := filepath.Join(opts.OutputDir, filepath.Base(imageFile)+".ppixmap")
		err = packedPixmap.Save(outputFile)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("---------------------------\n")
	fmt.Printf("unpackedSize=%v\n", unpackedSize/1024/1024)
	fmt.Printf("packedSize=%v\n", packedSize/1024/1024)
	fmt.Printf("unpackedSize/packedSize=%v\n", unpackedSize/packedSize)
}
