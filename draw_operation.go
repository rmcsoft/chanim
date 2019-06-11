package chanim

// DrawOperation is interface to encapsulate the drawing operation
type DrawOperation interface {
	Draw(paintEngine PaintEngine) error
}
