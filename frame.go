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
// be played to move to the destAnimation.
func (frame *Frame) GetSeriesForTransition(destAnimationName string) (string, bool) {
	for _, transition := range frame.Transitions {
		if transition.DestAnimationName == destAnimationName {
			return transition.FrameSeriesName, true
		}
	}
	return "", false
}
