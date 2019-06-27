package chanim

import "image"

type drawPackedPixmapOperation struct {
	top    image.Point
	pixmap *PackedPixmap
}

func (o *drawPackedPixmapOperation) Draw(paintEngine PaintEngine) error {
	return paintEngine.DrawPackedPixmap(o.top, o.pixmap)
}

// NewDrawPackedPixmapOperation creates an operation to draw the packed pixmap.
func NewDrawPackedPixmapOperation(top image.Point, pixmap *PackedPixmap) DrawOperation {
	return &drawPackedPixmapOperation{
		top:    top,
		pixmap: pixmap,
	}
}
