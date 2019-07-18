package chanim

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type animationState int

const (
	// Loop playback of current animation frames
	asPlayCurrentAnimation animationState = iota
	// Initialized animation change
	asInitChangeAnimation
	// Play frames of the current animation and look for a transition frame
	asFindTransitionFrame
	// Play a series of transition frames to move to the next animation
	asTransitionToNextAnimation
)

const defaultFrameRate = 25

// Animator implements character animation
type Animator struct {
	paintEngine    PaintEngine
	animations     Animations
	allFrameSeries []FrameSeries

	frameRate int

	mutex                 sync.Mutex
	isRunning             bool
	animationName         string
	nextAnimationName     *string
	animationChangedCond  *sync.Cond
	animationChangedError error

	state animationState

	playedFrames []Frame
	nextFrameNum int
	shownFrame   *Frame

	tryInitTransitionCounter int
}

// NewAnimator creats new Animator
func NewAnimator(paintEngine PaintEngine, animations Animations, allFrameSeries []FrameSeries) (*Animator, error) {

	animator := &Animator{
		paintEngine:    paintEngine,
		animations:     animations,
		allFrameSeries: allFrameSeries,
		frameRate:      defaultFrameRate,
		state:          asPlayCurrentAnimation,
	}
	animator.animationChangedCond = sync.NewCond(&animator.mutex)
	return animator, nil
}

// GetAnimationNames gets animation names
func (animator *Animator) GetAnimationNames() []string {
	animationNames := make([]string, 0)
	for _, animation := range animator.animations {
		animationNames = append(animationNames, animation.Name)
	}
	return animationNames
}

// Start drawing
func (animator *Animator) Start(initAnimationName string) error {
	animator.mutex.Lock()
	defer animator.mutex.Unlock()

	if animator.isRunning {
		return errors.New("Animator is already running")
	}

	err := animator.setAnimation(initAnimationName)
	if err != nil {
		return err
	}

	animator.isRunning = true
	go animator.doDraw()
	return nil
}

// Stop drawing
func (animator *Animator) Stop() {
	animator.mutex.Lock()
	animator.isRunning = false
	animator.mutex.Unlock()
}

// ChangeAnimation changes the current animation
func (animator *Animator) ChangeAnimation(nextAnimationName string) error {
	animator.mutex.Lock()
	defer animator.mutex.Unlock()

	if !animator.isRunning {
		return errors.New("Animator is not running")
	}

	if animator.nextAnimationName != nil || animator.state != asPlayCurrentAnimation {
		return errors.New("Animator is already making a animation change")
	}

	if animator.animationName == nextAnimationName {
		return nil
	}

	animator.nextAnimationName = &nextAnimationName
	animator.state = asInitChangeAnimation
	animator.animationChangedCond.Wait()
	animator.nextAnimationName = nil

	return animator.animationChangedError
}

func (animator *Animator) doDraw() {
	droppedFrameCount := 0
	showFrameDuration := time.Duration(1000/animator.frameRate) * time.Millisecond
	showNextFrameTime := time.Now()
	for {
		frame := animator.getCurremtFrame()
		if frame == nil {
			break
		}
		animator.shownFrame = frame

		showNextFrameTime = showNextFrameTime.Add(showFrameDuration)
		if time.Until(showNextFrameTime) <= 0 {
			droppedFrameCount++
			if droppedFrameCount%100 == 0 {
				fmt.Printf("Animator: the number of dropped frames: %v\n", droppedFrameCount)
			}
			continue
		}

		animator.paintEngine.Begin()
		if err := frame.Draw(animator.paintEngine); err != nil {
			panic(err)
		}
		animator.paintEngine.End()
		time.Sleep(time.Until(showNextFrameTime))
	}
}

func (animator *Animator) getCurremtFrame() *Frame {
	animator.mutex.Lock()
	defer animator.mutex.Unlock()

	if !animator.isRunning {
		return nil
	}

	if animator.state == asPlayCurrentAnimation {
		return animator.getCurrentAnimationFrame()
	}

	if animator.state == asInitChangeAnimation {
		animator.state = asFindTransitionFrame
		animator.tryInitTransitionCounter = 0
	}

	if animator.state == asFindTransitionFrame {
		return animator.tryInitTransitionToNextAnimation()
	}

	// ssTransitionToNextAnimation
	return animator.getCurrentTransitionFrame()
}

func (animator *Animator) getCurrentAnimationFrame() *Frame {
	frame := &animator.playedFrames[animator.nextFrameNum]
	animator.nextFrameNum = (animator.nextFrameNum + 1) % len(animator.playedFrames)
	return frame
}

func (animator *Animator) tryInitTransitionToNextAnimation() *Frame {
	animator.tryInitTransitionCounter++

	shownFrame := animator.shownFrame
	if shownFrame == nil || !shownFrame.IsTransitionFrame() {
		animator.checkFindTransitionFrameLooping()
		return animator.getCurrentAnimationFrame()
	}

	transitionFrameSeriesName, ok := shownFrame.GetSeriesForTransition(*animator.nextAnimationName)
	if !ok {
		animator.checkFindTransitionFrameLooping()
		return animator.getCurrentAnimationFrame()
	}

	var transitionFrames []Frame
	if transitionFrameSeriesName != "" {
		// To go to the next animation, need to play transition frames
		transitionFrameSeries := animator.findFrameSeriesByName(transitionFrameSeriesName)
		if transitionFrameSeries == nil {
			err := fmt.Errorf("Could't find a series of frames named '%s'", transitionFrameSeriesName)
			animator.finishChangeAnimation(err)
			return animator.getCurrentAnimationFrame()
		}

		transitionFrames = transitionFrameSeries.Frames
	}

	animator.state = asTransitionToNextAnimation
	animator.playedFrames = transitionFrames
	animator.nextFrameNum = 0
	return animator.getCurrentTransitionFrame()
}

func (animator *Animator) getCurrentTransitionFrame() *Frame {

	if animator.nextFrameNum < len(animator.playedFrames) {
		frame := &animator.playedFrames[animator.nextFrameNum]
		animator.nextFrameNum++
		return frame
	}

	animator.finishChangeAnimation(nil)
	return animator.getCurrentAnimationFrame()
}

func (animator *Animator) checkFindTransitionFrameLooping() {
	if animator.tryInitTransitionCounter >= len(animator.playedFrames) {
		err := fmt.Errorf("Could't find a transition frame for switch transition from '%s' to '%s'",
			animator.animationName, *animator.nextAnimationName)
		animator.finishChangeAnimation(err)
	}
}

func (animator *Animator) finishChangeAnimation(err error) {
	oldAnimation := animator.animationName

	if err == nil {
		err = animator.setAnimation(*animator.nextAnimationName)
	}

	if err != nil {
		animator.setAnimation(oldAnimation)
	}

	animator.animationChangedError = err
	animator.animationChangedCond.Signal()
}

func (animator *Animator) setAnimation(animationName string) error {
	animation := animator.findAnimationByName(animationName)
	if animation == nil {
		return fmt.Errorf("Could't find a animation named '%s'", animationName)
	}

	frameSeries := animator.findFrameSeriesByName(animation.FrameSeriesName)
	if frameSeries == nil {
		return fmt.Errorf("Could't find a series of frames named '%s'", animation.FrameSeriesName)
	}

	if len(frameSeries.Frames) == 0 {
		return fmt.Errorf("The frame series for animation '%s' is empty", animationName)
	}

	animator.animationName = animationName
	animator.playedFrames = frameSeries.Frames
	animator.nextFrameNum = 0
	animator.state = asPlayCurrentAnimation
	return nil
}

func (animator *Animator) findFrameSeriesByName(frameSeriesName string) *FrameSeries {
	for _, frameSeries := range animator.allFrameSeries {
		if frameSeries.Name == frameSeriesName {
			return &frameSeries
		}
	}
	return nil
}

func (animator *Animator) findAnimationByName(animationName string) *Animation {
	for _, animation := range animator.animations {
		if animation.Name == animationName {
			return &animation
		}
	}
	return nil
}
