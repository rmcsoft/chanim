package chanim

import "image"

type drawPixmapOperation struct {
	top    image.Point
	pixmap *Pixmap
}

func (o *drawPixmapOperation) Draw(paintEngine PaintEngine) error {
	return paintEngine.DrawPixmap(o.top, o.pixmap)
}

// NewDrawPixmapOperation creates an operation to draw the pixmap.
func NewDrawPixmapOperation(top image.Point, pixmap *Pixmap) DrawOperation {
	return &drawPixmapOperation{
		top:    top,
		pixmap: pixmap,
	}
}
