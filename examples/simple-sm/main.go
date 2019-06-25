package main

import (
	"fmt"
	"image"
	"reflect"

	"github.com/rmcsoft/chanim"
)

const (
	width           = 600
	height          = 600
	whiteSquareSize = 128
	pixFormat       = chanim.RGB16
	dx              = 5
	dy              = 5
)

func makeWhiteSquare() *chanim.Pixmap {
	bytePerLine := whiteSquareSize * chanim.GetPixelSize(pixFormat)
	pixels := make([]byte, whiteSquareSize*bytePerLine)

	for i := range pixels {
		pixels[i] = 0xFF
	}

	return &chanim.Pixmap{
		Data:        pixels,
		Width:       whiteSquareSize,
		Height:      whiteSquareSize,
		PixFormat:   pixFormat,
		BytePerLine: bytePerLine,
	}
}

var whiteSquare = makeWhiteSquare()

func reverseFrameSeries(frames []chanim.Frame) []chanim.Frame {
	reversed := []chanim.Frame{}
	for i := len(frames) - 1; i >= 0; i-- {
		reversed = append(reversed, frames[i])
	}
	return reversed
}

func makeVFrames(start int, end int) []chanim.Frame {
	cx := width/2 - whiteSquare.Width/2
	reverse := false
	if start > end {
		start, end = end, start
		reverse = true
	}

	frames := []chanim.Frame{}
	for y := start; y < end; y += dy {
		top := image.Point{cx, y}
		frames = append(frames,
			chanim.Frame{
				DrawOperations: []chanim.DrawOperation{
					chanim.NewDrawPixmapOperations(top, whiteSquare),
				},
			},
		)
	}

	if reverse {
		frames = reverseFrameSeries(frames)
	}

	return frames
}

func makeVFrameSeries() chanim.FrameSeries {
	cy := height/2 - whiteSquare.Height/2
	return chanim.FrameSeries{
		Name: "VFrameSeries",
		Frames: append(
			makeVFrames(cy, height),                 // Start from the center and go up
			makeVFrames(-whiteSquareSize+1, cy)...), // Come out from below and go to the center
	}
}

func makeHFrames(start int, end int) []chanim.Frame {
	cy := height/2 - whiteSquare.Height/2
	reverse := false
	if start > end {
		start, end = end, start
		reverse = true
	}

	frames := []chanim.Frame{}
	for x := start; x < end; x += dx {
		top := image.Point{x, cy}
		frames = append(frames,
			chanim.Frame{
				DrawOperations: []chanim.DrawOperation{
					chanim.NewDrawPixmapOperations(top, whiteSquare),
				},
			},
		)
	}

	if reverse {
		frames = reverseFrameSeries(frames)
	}

	return frames
}

func makeHFrameSeries() chanim.FrameSeries {
	cx := width/2 - whiteSquare.Width/2
	return chanim.FrameSeries{
		Name: "HFrameSeries",
		Frames: append(
			makeHFrames(cx, width),                  // We start from the center and go to the right edge
			makeHFrames(-whiteSquareSize+1, cx)...), // We leave to the left and go to the center
	}
}

func getWhiteSquareCoord(frame *chanim.Frame) image.Point {
	o := frame.DrawOperations[0]
	top := reflect.ValueOf(o).Elem().FieldByName("top")
	return image.Point{
		int(top.FieldByName("X").Int()),
		int(top.FieldByName("Y").Int()),
	}
}

func makeTransitionsFromHStatToVStat(hFrameSeries chanim.FrameSeries) []chanim.FrameSeries {
	res := []chanim.FrameSeries{}
	cx := width/2 - whiteSquare.Width/2
	for i := 0; i < len(hFrameSeries.Frames); i += 10 {
		frame := &hFrameSeries.Frames[i]

		name := fmt.Sprintf("H2V-%v", i)
		xWhiteSquare := getWhiteSquareCoord(frame).X
		transitionSeries := chanim.FrameSeries{
			Name:   name,
			Frames: makeHFrames(xWhiteSquare, cx), // Return the square to the center
		}
		res = append(res, transitionSeries)

		frame.Transitions = append(frame.Transitions, chanim.Transition{
			DestStateName:   "v",
			FrameSeriesName: name,
		})
	}

	return res
}

func makeTransitionsFromVStatToHStat(vFrameSeries chanim.FrameSeries) []chanim.FrameSeries {
	res := []chanim.FrameSeries{}
	cy := height/2 - whiteSquare.Height/2
	for i := 0; i < len(vFrameSeries.Frames); i += 10 {
		frame := &vFrameSeries.Frames[i]

		name := fmt.Sprintf("V2H-%v", i)
		yWhiteSquare := getWhiteSquareCoord(frame).Y
		transitionSeries := chanim.FrameSeries{
			Name:   name,
			Frames: makeVFrames(yWhiteSquare, cy), // Return the square to the center
		}
		res = append(res, transitionSeries)

		frame.Transitions = append(frame.Transitions, chanim.Transition{
			DestStateName:   "h",
			FrameSeriesName: name,
		})
	}

	return res
}

func makePaintEngine() chanim.PaintEngine {
	// paintEngine, err := chanim.NewSDLPaintEngine(width, height)
	top := image.Point{100, 100}
	viewport := image.Rect(top.X, top.Y, top.X+width, top.Y+height)
	paintEngine, err := chanim.NewKMSDRMPaintEngine(0, pixFormat, viewport)
	if err != nil {
		panic(err)
	}
	return paintEngine
}

func makeCharacterDrawer() *chanim.CharacterDrawer {
	hFrameSeries := makeHFrameSeries()
	vFrameSeries := makeVFrameSeries()
	allFrameSeries := []chanim.FrameSeries{
		hFrameSeries,
		vFrameSeries,
	}
	allFrameSeries = append(allFrameSeries, makeTransitionsFromHStatToVStat(hFrameSeries)...)
	allFrameSeries = append(allFrameSeries, makeTransitionsFromVStatToHStat(vFrameSeries)...)

	allStates := []chanim.State{
		chanim.State{
			Name:            "h",
			FrameSeriesName: hFrameSeries.Name,
		},
		chanim.State{
			Name:            "v",
			FrameSeriesName: vFrameSeries.Name,
		},
	}

	fmt.Printf("Available states:\n")
	for _, state := range allStates {
		fmt.Printf("\t%s\n", state.Name)
	}
	fmt.Println()

	paintEngine := makePaintEngine()
	drawer, err := chanim.NewCharacterDrawer(paintEngine, allStates, allFrameSeries)
	if err != nil {
		panic(err)
	}

	return drawer
}

func main() {
	drawer := makeCharacterDrawer()
	err := drawer.Start("h")
	if err != nil {
		panic(err)
	}

	for {
		newState := ""
		fmt.Printf("newState -> ")
		fmt.Scanf("%s", &newState)
		if newState == "" {
			break
		}

		err = drawer.ChangeState(newState)
		if err != nil {
			fmt.Println(err)
		}
	}
}
