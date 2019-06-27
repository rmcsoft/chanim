package main

import (
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/rmcsoft/chanim"
)

const (
	width     = 1024
	height    = 600
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
	top := image.Point{0, 0}
	viewport := image.Rect(top.X, top.Y, top.X+width, top.Y+height)
	paintEngine, err := chanim.NewKMSDRMPaintEngine(0, pixFormat, viewport)
	if err != nil {
		panic(err)
	}
	return paintEngine
}

func makeCharacterDrawer(intputDir string) *chanim.CharacterDrawer {
	frameSeries := loadFrameseries(intputDir)
	allFrameSeries := []chanim.FrameSeries{frameSeries}
	allStates := []chanim.State{
		chanim.State{
			Name:            "init",
			FrameSeriesName: frameSeries.Name,
		},
	}

	paintEngine := makePaintEngine()
	drawer, err := chanim.NewCharacterDrawer(paintEngine, allStates, allFrameSeries)
	if err != nil {
		panic(err)
	}

	return drawer
}

func main() {
	intputDir := os.Args[1]
	drawer := makeCharacterDrawer(intputDir)
	err := drawer.Start("init")
	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}
