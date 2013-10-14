package cocoa

// #cgo CFLAGS: -x objective-c -fobjc-arc
// #cgo LDFLAGS: -framework Cocoa -framework OpenGL -framework QuartzCore
//
// #include <stdlib.h>
// #include "input.h"
//
// void Run(size_t width, size_t height, size_t scale, const char* title);
//
import "C"
import (
	"github.com/hajimehoshi/go-ebiten"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl"
	"time"
	"unsafe"
)

type UI struct {
	screenWidth    int
	screenHeight   int
	screenScale    int
	graphicsDevice *opengl.Device
	initializing   chan ebiten.Game
	initialized    chan ebiten.Game
	updating       chan ebiten.Game
	updated        chan ebiten.Game
	input          chan ebiten.InputState
}

var currentUI *UI

//export ebiten_EbitenOpenGLView_Initialized
func ebiten_EbitenOpenGLView_Initialized() {
	if currentUI.graphicsDevice != nil {
		panic("The graphics device is already initialized")
	}

	currentUI.graphicsDevice = opengl.NewDevice(
		currentUI.screenWidth,
		currentUI.screenHeight,
		currentUI.screenScale)
	currentUI.graphicsDevice.Init()

	game := <-currentUI.initializing
	game.Init(currentUI.graphicsDevice.TextureFactory())
	currentUI.initialized <- game
}

//export ebiten_EbitenOpenGLView_Updating
func ebiten_EbitenOpenGLView_Updating() {
	game := <-currentUI.updating
	currentUI.graphicsDevice.Update(game.Draw)
	currentUI.updated <- game
}

//export ebiten_EbitenOpenGLView_InputUpdated
func ebiten_EbitenOpenGLView_InputUpdated(inputType C.InputType, cx, cy C.int) {
	if inputType == C.InputTypeMouseUp {
		currentUI.input <- ebiten.InputState{-1, -1}
		return
	}

	x, y := int(cx), int(cy)
	x /= currentUI.screenScale
	y /= currentUI.screenScale
	if x < 0 {
		x = 0
	} else if currentUI.screenWidth <= x {
		x = currentUI.screenWidth - 1
	}
	if y < 0 {
		y = 0
	} else if currentUI.screenHeight <= y {
		y = currentUI.screenHeight - 1
	}
	currentUI.input <- ebiten.InputState{x, y}
}

func Run(game ebiten.Game, screenWidth, screenHeight, screenScale int,
	title string) {
	currentUI = &UI{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
		initializing: make(chan ebiten.Game),
		initialized:  make(chan ebiten.Game),
		updating:     make(chan ebiten.Game),
		updated:      make(chan ebiten.Game),
		input:        make(chan ebiten.InputState),
	}

	go func() {
		frameTime := time.Duration(
			int64(time.Second) / int64(ebiten.FPS))
		tick := time.Tick(frameTime)
		gameContext := &GameContext{
			screenWidth:  screenWidth,
			screenHeight: screenHeight,
			inputState:   ebiten.InputState{-1, -1},
		}
		currentUI.initializing <- game
		game = <-currentUI.initialized
		for {
			select {
			case gameContext.inputState = <-currentUI.input:
			case <-tick:
				game.Update(gameContext)
			case currentUI.updating <- game:
				game = <-currentUI.updated
			}
		}
	}()

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.Run(C.size_t(screenWidth),
		C.size_t(screenHeight),
		C.size_t(screenScale),
		cTitle)
}

type GameContext struct {
	screenWidth  int
	screenHeight int
	inputState   ebiten.InputState
}

func (context *GameContext) ScreenWidth() int {
	return context.screenWidth
}

func (context *GameContext) ScreenHeight() int {
	return context.screenHeight
}

func (context *GameContext) InputState() ebiten.InputState {
	return context.inputState
}
