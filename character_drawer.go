package chanim

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type subState int

const (
	// Loop playback of current state frames
	ssPlayCurrentState subState = iota
	// Initialized state change
	ssInitChangeState
	// Play frames of the current state and look for a transition frame
	ssFindTransitionFrame
	// Play a series of transition frames to move to the next state
	ssTransitionToNextState
)

const defaultFrameRate = 25

// CharacterDrawer implements character animation
type CharacterDrawer struct {
	paintEngine    PaintEngine
	allStates      []State
	allFrameSeries []FrameSeries

	frameRate int

	mutex             sync.Mutex
	isRunning         bool
	stateName         string
	nextStateName     *string
	stateChangedCond  *sync.Cond
	stateChangedError error

	subState subState

	playedFrames    []Frame
	currentFrameNum int

	startFindTransitionFrameNum int
}

// NewCharacterDrawer creats new CharacterDrawer
func NewCharacterDrawer(paintEngine PaintEngine, allStates []State, allFrameSeries []FrameSeries) (*CharacterDrawer, error) {

	drawer := &CharacterDrawer{
		paintEngine:    paintEngine,
		allStates:      allStates,
		allFrameSeries: allFrameSeries,
		frameRate:      defaultFrameRate,
		subState:       ssPlayCurrentState,
	}
	drawer.stateChangedCond = sync.NewCond(&drawer.mutex)
	return drawer, nil
}

// Start drawing
func (drawer *CharacterDrawer) Start(initStateName string) error {
	drawer.mutex.Lock()
	defer drawer.mutex.Unlock()

	if drawer.isRunning {
		return errors.New("CharacterDrawer is already running")
	}

	err := drawer.setState(initStateName)
	if err != nil {
		return err
	}

	drawer.isRunning = true
	go drawer.doDraw()
	return nil
}

// Stop drawing
func (drawer *CharacterDrawer) Stop() {
	drawer.mutex.Lock()
	drawer.isRunning = false
	drawer.mutex.Unlock()
}

// ChangeState changes the current state
func (drawer *CharacterDrawer) ChangeState(nextStateName string) error {
	drawer.mutex.Lock()
	defer drawer.mutex.Unlock()

	if !drawer.isRunning {
		return errors.New("CharacterDrawer is not running")
	}

	if drawer.nextStateName != nil || drawer.subState != ssPlayCurrentState {
		return errors.New("CharacterDrawer is already making a state change")
	}

	if drawer.stateName == nextStateName {
		return nil
	}

	drawer.nextStateName = &nextStateName
	drawer.subState = ssInitChangeState
	drawer.stateChangedCond.Wait()
	drawer.nextStateName = nil

	return drawer.stateChangedError
}

func (drawer *CharacterDrawer) doDraw() {
	droppedFrameCount := 0
	showFrameDuration := time.Duration(1000/drawer.frameRate) * time.Millisecond
	showNextFrameTime := time.Now()
	for {
		frame := drawer.getCurremtFrame()
		if frame == nil {
			break
		}

		showNextFrameTime = showNextFrameTime.Add(showFrameDuration)
		if time.Until(showNextFrameTime) <= 0 {
			droppedFrameCount++
			if droppedFrameCount%100 == 0 {
				fmt.Printf("CharacterDrawer: the number of dropped frames: %v\n", droppedFrameCount)
			}
			continue
		}

		drawer.paintEngine.Begin()
		frame.Draw(drawer.paintEngine)
		drawer.paintEngine.End()
		time.Sleep(time.Until(showNextFrameTime))
	}
}

func (drawer *CharacterDrawer) getCurremtFrame() *Frame {
	drawer.mutex.Lock()
	defer drawer.mutex.Unlock()

	if !drawer.isRunning {
		return nil
	}

	if drawer.subState == ssPlayCurrentState {
		return drawer.getCurrentStateFrame()
	}

	if drawer.subState == ssInitChangeState {
		drawer.subState = ssFindTransitionFrame
		drawer.startFindTransitionFrameNum = drawer.currentFrameNum
	}

	if drawer.subState == ssFindTransitionFrame {
		return drawer.tryInitTransitionToNextState()
	}

	// ssTransitionToNextState
	return drawer.getCurrentTransitionFrame()
}

func (drawer *CharacterDrawer) getCurrentStateFrame() *Frame {
	frame := &drawer.playedFrames[drawer.currentFrameNum]
	drawer.currentFrameNum = (drawer.currentFrameNum + 1) % len(drawer.playedFrames)
	return frame
}

func (drawer *CharacterDrawer) tryInitTransitionToNextState() *Frame {
	frame := drawer.getCurrentStateFrame()

	if !frame.IsTransitionFrame() {
		drawer.checkFindTransitionFrameLooping()
		return frame
	}

	transitionFrameSeriesName := frame.GetSeriesForTransition(*drawer.nextStateName)
	if transitionFrameSeriesName == nil {
		drawer.checkFindTransitionFrameLooping()
		return frame
	}

	transitionFrameSeries := drawer.findFrameSeriesByName(*transitionFrameSeriesName)
	if transitionFrameSeries == nil {
		err := fmt.Errorf("Could't find a series of frames named '%s'", *transitionFrameSeriesName)
		drawer.finishChangeState(err)
		return frame
	}

	drawer.subState = ssTransitionToNextState
	drawer.playedFrames = transitionFrameSeries.Frames
	drawer.currentFrameNum = 0

	return frame
}

func (drawer *CharacterDrawer) getCurrentTransitionFrame() *Frame {

	if drawer.currentFrameNum < len(drawer.playedFrames) {
		frame := &drawer.playedFrames[drawer.currentFrameNum]
		drawer.currentFrameNum++
		return frame
	}

	drawer.finishChangeState(nil)
	return drawer.getCurrentStateFrame()
}

func (drawer *CharacterDrawer) checkFindTransitionFrameLooping() {
	if drawer.currentFrameNum == drawer.startFindTransitionFrameNum {
		err := fmt.Errorf("Could't find a transition frame for swithe transition from '%s' to '%s'",
			drawer.stateName, *drawer.nextStateName)
		drawer.finishChangeState(err)
	}
}

func (drawer *CharacterDrawer) finishChangeState(err error) {
	oldState := drawer.stateName

	if err == nil {
		err = drawer.setState(*drawer.nextStateName)
	}

	if err != nil {
		drawer.setState(oldState)
	}

	drawer.stateChangedError = err
	drawer.stateChangedCond.Signal()
}

func (drawer *CharacterDrawer) setState(stateName string) error {
	state := drawer.findStateByName(stateName)
	if state == nil {
		return fmt.Errorf("Could't find a state named '%s'", stateName)
	}

	frameSeries := drawer.findFrameSeriesByName(state.FrameSeriesName)
	if frameSeries == nil {
		return fmt.Errorf("Could't find a series of frames named '%s'", state.FrameSeriesName)
	}

	if len(frameSeries.Frames) == 0 {
		return fmt.Errorf("The frame series for state '%s' is empty", stateName)
	}

	drawer.stateName = stateName
	drawer.playedFrames = frameSeries.Frames
	drawer.currentFrameNum = 0
	drawer.subState = ssPlayCurrentState
	return nil
}

func (drawer *CharacterDrawer) findFrameSeriesByName(frameSeriesName string) *FrameSeries {
	for _, frameSeries := range drawer.allFrameSeries {
		if frameSeries.Name == frameSeriesName {
			return &frameSeries
		}
	}
	return nil
}

func (drawer *CharacterDrawer) findStateByName(stateName string) *State {
	for _, state := range drawer.allStates {
		if state.Name == stateName {
			return &state
		}
	}
	return nil
}
