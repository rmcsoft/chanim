package chanim

import (
	"image"
)

type nullPaintEngine struct {
}

// NullPaintEngine returns null paint engine
func NullPaintEngine() PaintEngine {
	return nullPaintEngine{}
}

func (nullPaintEngine) GetWidth() int {
	return 0
}

func (nullPaintEngine) GetHeight() int {
	return 0
}

func (nullPaintEngine) Begin() error {
	return nil
}

func (nullPaintEngine) Clear(rect image.Rectangle) error {
	return nil
}

func (nullPaintEngine) DrawPixmap(top image.Point, pixmap *Pixmap) error {
	return nil
}

func (nullPaintEngine) DrawPackedPixmap(top image.Point, packedPixmap *PackedPixmap) error {
	return nil
}

func (nullPaintEngine) End() error {
	return nil
}
