package chanim

import "image"

type drawPixmapOperations struct {
	top    image.Point
	pixmap Pixmap
}

func (o *drawPixmapOperations) Draw(paintEngine PaintEngine) error {
	return paintEngine.DrawPixmap(o.top, &o.pixmap)
}
