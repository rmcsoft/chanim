package chanim

import (
	"image"
	"sync"

	"github.com/veandco/go-sdl2/sdl"
)

var mutexSdlInit = sync.Mutex{}
var sdlInited = false

func initSdl() {
	mutexSdlInit.Lock()
	defer mutexSdlInit.Unlock()

	if !sdlInited {
		sdl.Init(sdl.INIT_VIDEO)
		sdlInited = true
	}
}

type sdlPaintEngine struct {
	window   *sdl.Window
	renderer *sdl.Renderer
}

// NewSDLPaintEngine creates NewSDLPaintEngine
func NewSDLPaintEngine(width int, height int) (PaintEngine, error) {
	window, renderer, err := sdl.CreateWindowAndRenderer(int32(width), int32(height), 0)
	if err != nil {
		return nil, err
	}

	return &sdlPaintEngine{window, renderer}, nil
}

func (p *sdlPaintEngine) Begin() error {
	return p.renderer.Clear()
}

func (p *sdlPaintEngine) Clear(rect image.Rectangle) error {
	sdlRect := sdl.Rect{
		X: int32(rect.Min.X),
		Y: int32(rect.Min.Y),
		W: int32(rect.Dx()),
		H: int32(rect.Dy()),
	}
	return p.renderer.FillRect(&sdlRect)
}

func (p *sdlPaintEngine) DrawPixmap(top image.Point, pixmap *Pixmap) error {
	sdlPixFormat, err := pixelFormatToSDL(pixmap.PixFormat)
	if err != nil {
		return err
	}

	texture, err := p.renderer.CreateTexture(sdlPixFormat, sdl.TEXTUREACCESS_STREAMING,
		int32(pixmap.Width), int32(pixmap.Height))
	if err != nil {
		return err
	}
	defer texture.Destroy()

	texturePixels, textureBytePerLine, err := texture.Lock(nil)
	if err != nil {
		return err
	}

	rowSize := pixmap.Width * GetPixelSize(pixmap.PixFormat)
	for rowNum := 0; rowNum < pixmap.Height; rowNum++ {
		pixmapOffset := rowNum * pixmap.BytePerLine
		pixmapRow := pixmap.Data[pixmapOffset : pixmapOffset+rowSize]
		textureOffset := rowNum * textureBytePerLine
		textureRow := texturePixels[textureOffset : textureOffset+rowSize]
		copy(textureRow, pixmapRow)
	}
	texture.Unlock()

	sdlRect := sdl.Rect{
		X: int32(top.X),
		Y: int32(top.Y),
		W: int32(pixmap.Width),
		H: int32(pixmap.Height),
	}
	return p.renderer.Copy(texture, nil, &sdlRect)
}

func (p *sdlPaintEngine) End() error {
	p.renderer.Present()
	return nil
}
