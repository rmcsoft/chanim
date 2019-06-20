package chanim

import "image"

type drawPixmapOperations struct {
	top    image.Point
	pixmap *Pixmap
}

func (o *drawPixmapOperations) Draw(paintEngine PaintEngine) error {
	return paintEngine.DrawPixmap(o.top, o.pixmap)
}

// NewDrawPixmapOperations creates an operation to draw the pixmap.
func NewDrawPixmapOperations(top image.Point, pixmap *Pixmap) DrawOperation {
	return &drawPixmapOperations{
		top:    top,
		pixmap: pixmap,
	}
}
