package chanim

// Frame contains a set of operations for drawing.
type Frame struct {
	DrawOperations []DrawOperation
	Transitions    []Transition
}

// Draw draws a frame delta.
func (frame *Frame) Draw(paintEngine PaintEngine) error {
	for _, drawOperation := range frame.DrawOperations {
		err := drawOperation.Draw(paintEngine)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsTransitionFrame checks is frame transitional.
func (frame *Frame) IsTransitionFrame() bool {
	return frame.Transitions != nil
}

// GetSeriesForTransition returns the name of the series of frames that shuld
// be played to move to the destState.
func (frame *Frame) GetSeriesForTransition(destStateName string) *string {
	for _, transition := range frame.Transitions {
		if transition.DestStateName == destStateName {
			return &transition.FrameSeriesName
		}
	}
	return nil
}
