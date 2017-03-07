// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/window"
	"sort"
)

type Root struct {
	Panel                            // embedded panel
	core.TimerManager                // embedded TimerManager
	gs                *gls.GLS       // OpenGL state
	win               window.IWindow // Window
	stopPropagation   int            // stop event propagation bitmask
	keyFocus          IPanel         // current child panel with key focus
	mouseFocus        IPanel         // current child panel with mouse focus
	scrollFocus       IPanel         // current child panel with scroll focus
	targets           listPanelZ     // preallocated list of target panels
}

const (
	StopGUI = 0x01             // Stop event propagation to GUI
	Stop3D  = 0x02             // Stop event propagation to 3D
	StopAll = StopGUI | Stop3D // Stop event propagation
)

// NewRoot creates and returns a pointer to a gui root panel for the specified window
func NewRoot(gs *gls.GLS, win window.IWindow) *Root {

	r := new(Root)
	r.gs = gs
	r.win = win
	r.Panel.Initialize(0, 0)
	r.TimerManager.Initialize()
	// for optimization, sets this root panel as not renderable as in most cases
	// it is used only as a container
	r.SetRenderable(false)
	// Subscribe to window events
	r.SubscribeWin()
	r.targets = []IPanel{}
	return r
}

// SubscribeWin subscribes this root panel to window events
func (r *Root) SubscribeWin() {

	r.win.Subscribe(window.OnKeyUp, r.onKey)
	r.win.Subscribe(window.OnKeyDown, r.onKey)
	r.win.Subscribe(window.OnChar, r.onChar)
	r.win.Subscribe(window.OnMouseUp, r.onMouse)
	r.win.Subscribe(window.OnMouseDown, r.onMouse)
	r.win.Subscribe(window.OnCursor, r.onCursor)
	r.win.Subscribe(window.OnScroll, r.onScroll)
	r.win.Subscribe(window.OnWindowSize, r.onWindowSize)
	r.win.Subscribe(window.OnFrame, r.onFrame)
}

// Add adds the specified panel to the root container list of children
func (r *Root) Add(ipan IPanel) {

	r.Panel.Add(ipan)
	ipan.GetNode().SetParent(r)
}

// SetKeyFocus sets the panel which will receive all keyboard events
// Passing nil will remove the focus (if any)
func (r *Root) SetKeyFocus(ipan IPanel) {

	if r.keyFocus != nil {
		// If this panel is already in focus, nothing to do
		if ipan != nil {
			if r.keyFocus.GetPanel() == ipan.GetPanel() {
				return
			}
		}
		r.keyFocus.LostKeyFocus()
	}
	r.keyFocus = ipan
}

// ClearKeyFocus clears the key focus panel (if any) without
// calling LostKeyFocus() for previous focused panel
func (r *Root) ClearKeyFocus() {

	r.keyFocus = nil
}

// SetMouseFocus sets the panel which will receive all mouse events
// Passing nil will restore the default event processing
func (r *Root) SetMouseFocus(ipan IPanel) {

	r.mouseFocus = ipan
}

// SetScrollFocus sets the panel which will receive all scroll events
// Passing nil will restore the default event processing
func (r *Root) SetScrollFocus(ipan IPanel) {

	r.scrollFocus = ipan
}

// HasKeyFocus checks if the specified panel has the key focus
func (r *Root) HasKeyFocus(ipan IPanel) bool {

	if r.keyFocus == nil {
		return false
	}
	if r.keyFocus.GetPanel() == ipan.GetPanel() {
		return true
	}
	return false
}

// HasMouseFocus checks if the specified panel has the mouse focus
func (r *Root) HasMouseFocus(ipan IPanel) bool {

	if r.mouseFocus == nil {
		return false
	}
	if r.mouseFocus.GetPanel() == ipan.GetPanel() {
		return true
	}
	return false
}

// StopPropagation stops the propagation of the current event
// to outside the root panel (for example the 3D camera)
func (r *Root) StopPropagation(events int) {

	r.stopPropagation |= events
}

// SetCursorNormal sets the cursor of the associated window to
// standard type
func (r *Root) SetCursorNormal() {

	r.win.SetStandardCursor(window.ArrowCursor)
}

// SetCursorDrag sets the cursor of the associated window to
// drag type
func (r *Root) SetCursorDrag() {

	r.win.SetStandardCursor(window.HandCursor)
}

// SetCursorHResize sets the cursor of the associated window to
// horizontal resize type
func (r *Root) SetCursorHResize() {

	r.win.SetStandardCursor(window.HResizeCursor)
}

// SetCursorVResize sets the cursor of the associated window to
// vertical resize type
func (r *Root) SetCursorVResize() {

	r.win.SetStandardCursor(window.VResizeCursor)
}

// onKey is called when key events are received
func (r *Root) onKey(evname string, ev interface{}) {

	// If no panel has the key focus, nothing to do
	if r.keyFocus == nil {
		return
	}
	// Dispatch window.KeyEvent to focused panel subscribers
	r.stopPropagation = 0
	r.keyFocus.GetPanel().Dispatch(evname, ev)
	// If requested, stopj propagation of event outside the root gui
	if (r.stopPropagation & Stop3D) != 0 {
		r.win.CancelDispatch()
	}
}

// onChar is called when char events are received
func (r *Root) onChar(evname string, ev interface{}) {

	// If no panel has the key focus, nothing to do
	if r.keyFocus == nil {
		return
	}
	// Dispatch window.CharEvent to focused panel subscribers
	r.stopPropagation = 0
	r.keyFocus.GetPanel().Dispatch(evname, ev)
	// If requested, stopj propagation of event outside the root gui
	if (r.stopPropagation & Stop3D) != 0 {
		r.win.CancelDispatch()
	}
}

// onMouse is called when mouse button events are received
func (r *Root) onMouse(evname string, ev interface{}) {

	mev := ev.(*window.MouseEvent)
	r.sendPanels(mev.Xpos, mev.Ypos, evname, ev)
}

// onCursor is called when (mouse) cursor events are received
func (r *Root) onCursor(evname string, ev interface{}) {

	cev := ev.(*window.CursorEvent)
	r.sendPanels(cev.Xpos, cev.Ypos, evname, ev)
}

// sendPanel sends mouse or cursor event to focused panel or panels
// which contains the specified screen position
func (r *Root) sendPanels(x, y float32, evname string, ev interface{}) {

	// If there is panel with MouseFocus send only to this panel
	if r.mouseFocus != nil {
		r.mouseFocus.GetPanel().Dispatch(evname, ev)
		if (r.stopPropagation & Stop3D) != 0 {
			r.win.CancelDispatch()
		}
		return
	}

	// Clear list of panels which contains the mouse position
	r.targets = r.targets[0:0]

	// checkPanel checks recursively if the specified panel and
	// any of its child contains the mouse position
	var checkPanel func(ipan IPanel)
	checkPanel = func(ipan IPanel) {
		pan := ipan.GetPanel()
		// If panel not visible or not enabled, ignore
		if !pan.Visible() || !pan.Enabled() {
			return
		}
		// Checks if this panel contains the mouse position
		found := pan.ContainsPosition(x, y)
		if found {
			r.targets = append(r.targets, ipan)
		} else {
			// If OnCursorEnter previously sent, sends OnMouseLeave
			if pan.cursorEnter {
				pan.Dispatch(OnCursorLeave, ev)
				pan.cursorEnter = false
			}
			// If mouse button was pressed, sends event informing mouse down outside of the panel
			if evname == OnMouseDown {
				pan.Dispatch(OnMouseOut, ev)
			}
		}
		// Checks if any of its children also contains the position
		for _, child := range pan.Children() {
			ipan := child.(IPanel)
			checkPanel(ipan)
		}
	}

	// Checks all children of this root node
	for _, iobj := range r.Node.Children() {
		ipan, ok := iobj.(IPanel)
		if !ok {
			continue
		}
		checkPanel(ipan)
	}

	// No panels found
	if len(r.targets) == 0 {
		// If event is mouse click, removes the keyboard focus
		if evname == OnMouseDown {
			r.SetKeyFocus(nil)
		}
		return
	}

	// Sorts panels by absolute Z with the most foreground panels first
	// and sends event to all panels or until a stop is requested
	sort.Sort(r.targets)
	r.stopPropagation = 0

	// Send events to panels
	for _, ipan := range r.targets {
		ipan.SetRoot(r)
		pan := ipan.GetPanel()
		// Cursor position event
		if evname == OnCursor {
			pan.Dispatch(evname, ev)
			if !pan.cursorEnter {
				pan.Dispatch(OnCursorEnter, ev)
				pan.cursorEnter = true
			}
			// Mouse button event
		} else {
			pan.Dispatch(evname, ev)
		}
		if (r.stopPropagation & StopGUI) != 0 {
			break
		}
	}

	// Stops propagation of event outside the root gui
	if (r.stopPropagation & Stop3D) != 0 {
		r.win.CancelDispatch()
	}
}

// onScroll is called when scroll events are received and
// is responsible to dispatch them to child panels.
func (r *Root) onScroll(evname string, ev interface{}) {

	// If no panel with the scroll focus, nothing to do
	if r.scrollFocus == nil {
		return
	}
	// Dispatch event to panel with scroll focus
	r.scrollFocus.GetPanel().Dispatch(evname, ev)

	// Stops propagation of event outside the root gui
	if (r.stopPropagation & Stop3D) != 0 {
		r.win.CancelDispatch()
	}
}

// onSize is called when window size events are received
func (r *Root) onWindowSize(evname string, ev interface{}) {

	// Sends event only to immediate children
	for _, ipan := range r.Children() {
		ipan.(IPanel).GetPanel().Dispatch(evname, ev)
	}
}

// onFrame is called when window finished swapping frame buffers
func (r *Root) onFrame(evname string, ev interface{}) {

	r.TimerManager.ProcessTimers()
}

// For sorting panels by Z coordinate
type listPanelZ []IPanel

func (p listPanelZ) Len() int      { return len(p) }
func (p listPanelZ) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p listPanelZ) Less(i, j int) bool {

	iz := p[i].GetPanel().pospix.Z
	jz := p[j].GetPanel().pospix.Z
	return iz < jz
}