package chanim

import (
	"image"

	"github.com/veandco/go-sdl2/sdl"
)

func init() {
	sdl.Init(sdl.INIT_VIDEO)
}

// SDLPaintEngine is PaintEngine for kmsdrm
type SDLPaintEngine struct {
	window   *sdl.Window
	renderer *sdl.Renderer
}

// NewSDLPaintEngine creates NewSDLPaintEngine
func NewSDLPaintEngine(width int, height int) (*SDLPaintEngine, error) {
	window, renderer, err := sdl.CreateWindowAndRenderer(int32(width), int32(height), 0)
	if err != nil {
		return nil, err
	}

	return &SDLPaintEngine{window, renderer}, nil
}

// Begin begins paint
func (p *SDLPaintEngine) Begin() error {
	return nil
}

// Clear clears the rectangle
func (p *SDLPaintEngine) Clear(rect image.Rectangle) error {
	sdlRect := sdl.Rect{
		X: int32(rect.Min.X),
		Y: int32(rect.Min.Y),
		W: int32(rect.Dx()),
		H: int32(rect.Dy()),
	}
	return p.renderer.FillRect(&sdlRect)
}

// DrawPixmap draws the Pixmap
func (p *SDLPaintEngine) DrawPixmap(top image.Point, pixmap *Pixmap) error {
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
		pixmapRow := pixmap.Data[rowNum*pixmap.BytePerLine : rowSize]
		textureRow := texturePixels[rowNum*textureBytePerLine : rowSize]
		copy(pixmapRow, textureRow)
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

// End ends paint
func (p *SDLPaintEngine) End() error {
	p.renderer.Present()
	return nil
}
