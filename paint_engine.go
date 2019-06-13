package chanim

import (
	"image"
)

// PaintEngine is the interface definition for drawing
type PaintEngine interface {
	Begin() error
	Clear(rect image.Rectangle) error
	DrawPixmap(top image.Point, pixmap *Pixmap) error
	End() error
}