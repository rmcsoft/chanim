package main

import (
	"image"
	"os"
	"path/filepath"

	"github.com/rmcsoft/chanim"
)

const (
	pixFormat = chanim.RGB16
)

func loadFrameseries(intputDir string) chanim.FrameSeries {

	inputFiles, err := filepath.Glob(filepath.Join(intputDir, "*.ppixmap"))
	if err != nil {
		panic(err)
	}

	frames := make([]chanim.Frame, 0)
	for _, inputFile := range inputFiles {
		ppixmap, err := chanim.LoadPackedPixmap(inputFile)
		if err != nil {
			panic(err)
		}

		frame := chanim.Frame{
			DrawOperations: []chanim.DrawOperation{
				chanim.NewDrawPackedPixmapOperation(image.Point{0, 0}, ppixmap),
			},
		}
		frames = append(frames, frame)
	}

	return chanim.FrameSeries{
		Name:   "TestFrameSeries",
		Frames: frames,
	}
}

func makePaintEngine() chanim.PaintEngine {
	// paintEngine, err := chanim.NewSDLPaintEngine(width, height)
	paintEngine, err := chanim.NewKMSDRMPaintEngine(0, pixFormat)
	if err != nil {
		panic(err)
	}
	return paintEngine
}

func makeAnimator(intputDir string) *chanim.Animator {
	frameSeries := loadFrameseries(intputDir)
	allFrameSeries := []chanim.FrameSeries{frameSeries}
	animations := chanim.Animations{
		chanim.Animation{
			Name:            "init",
			FrameSeriesName: frameSeries.Name,
		},
	}

	paintEngine := makePaintEngine()
	animator, err := chanim.NewAnimator(paintEngine, animations, allFrameSeries)
	if err != nil {
		panic(err)
	}

	return animator
}

func main() {
	intputDir := os.Args[1]
	animator := makeAnimator(intputDir)
	err := animator.Start("init")
	if err != nil {
		panic(err)
	}

	select {}
}
