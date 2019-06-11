package chanim

import "image"

type clearOperation struct {
	rect image.Rectangle
}

func (o *clearOperation) Draw(paintEngine PaintEngine) error {
	return paintEngine.Clear(o.rect)
}

// NewClearDrawOperation creates an operation to clear the specified rectangle.
func NewClearDrawOperation(rect image.Rectangle) DrawOperation {
	return &clearOperation{rect}
}
