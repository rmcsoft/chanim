package chanim

// Frame is a frame that is not drawn completely, but partially,
// given the previous frame.
type Frame struct {
	drawOperations []DrawOperation
	transitions    []Transition
}

// Draw draws a frame delta.
func (frame *Frame) Draw(paintEngine PaintEngine) error {
	for _, drawOperation := range frame.drawOperations {
		err := drawOperation.Draw(paintEngine)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsTransitionFrame checks is frame transitional.
func (frame *Frame) IsTransitionFrame() bool {
	return frame.transitions != nil
}

// GetSeriesForTransition returns the name of the series of frames that shuld
// be played to move to the destState.
func (frame *Frame) GetSeriesForTransition(destStateName string) *string {
	if frame.transitions == nil {
		for _, transition := range frame.transitions {
			if transition.DestStateName == destStateName {
				return &transition.FrameSeriesName
			}
		}
	}

	return nil
}
