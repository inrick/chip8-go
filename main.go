package main

import (
	"chip8-go/chip8"
	"fmt"
	"github.com/go-gl/gl/v4.5-compatibility/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"os"
	"runtime"
)

const renderScale = 15

func drawPoint(x, y int) {
	xf := float32(x)
	yf := float32(y)
	gl.Begin(gl.QUADS)
	gl.Vertex2f(xf, yf)
	gl.Vertex2f(xf+1, yf)
	gl.Vertex2f(xf+1, yf+1)
	gl.Vertex2f(xf, yf+1)
	gl.End()
}

func render(c8 *chip8.Chip8) {
	gl.Color3f(.85, .85, .85)
	for i := range c8.Gfx {
		for j := range c8.Gfx[i] {
			if c8.Gfx[i][j] == 1 {
				drawPoint(i, j)
			}
		}
	}
}

func resizeHandler(w *glfw.Window, width, height int) {
	gl.ClearColor(0., 0., 0., 0)
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, chip8.DisplayWidth, chip8.DisplayHeight, 0, 0, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.Viewport(0, 0, int32(width), int32(height))
}

func keyHandler(c8 *chip8.Chip8) glfw.KeyCallback {
	return func(
		window *glfw.Window, key glfw.Key, scancode int,
		action glfw.Action, mods glfw.ModifierKey) {
		// Keypad    =>  Keyboard
		// |1|2|3|C|     |1|2|3|4|
		// |4|5|6|D|     |Q|W|E|R|
		// |7|8|9|E|     |A|S|D|F|
		// |A|0|B|F|     |Z|X|C|V|
		switch action {
		case glfw.Press:
			switch key {
			case glfw.Key1:
				c8.Key[0x1] = true
			case glfw.Key2:
				c8.Key[0x2] = true
			case glfw.Key3:
				c8.Key[0x3] = true
			case glfw.Key4:
				c8.Key[0xC] = true
			case glfw.KeyQ:
				c8.Key[0x4] = true
			case glfw.KeyW:
				c8.Key[0x5] = true
			case glfw.KeyE:
				c8.Key[0x6] = true
			case glfw.KeyR:
				c8.Key[0xD] = true
			case glfw.KeyA:
				c8.Key[0x7] = true
			case glfw.KeyS:
				c8.Key[0x8] = true
			case glfw.KeyD:
				c8.Key[0x9] = true
			case glfw.KeyF:
				c8.Key[0xE] = true
			case glfw.KeyZ:
				c8.Key[0xA] = true
			case glfw.KeyX:
				c8.Key[0x0] = true
			case glfw.KeyC:
				c8.Key[0xB] = true
			case glfw.KeyV:
				c8.Key[0xF] = true
			case glfw.KeyEscape:
				window.SetShouldClose(true)
			}
		case glfw.Release:
			switch key {
			case glfw.Key1:
				c8.Key[0x1] = false
			case glfw.Key2:
				c8.Key[0x2] = false
			case glfw.Key3:
				c8.Key[0x3] = false
			case glfw.Key4:
				c8.Key[0xC] = false
			case glfw.KeyQ:
				c8.Key[0x4] = false
			case glfw.KeyW:
				c8.Key[0x5] = false
			case glfw.KeyE:
				c8.Key[0x6] = false
			case glfw.KeyR:
				c8.Key[0xD] = false
			case glfw.KeyA:
				c8.Key[0x7] = false
			case glfw.KeyS:
				c8.Key[0x8] = false
			case glfw.KeyD:
				c8.Key[0x9] = false
			case glfw.KeyF:
				c8.Key[0xE] = false
			case glfw.KeyZ:
				c8.Key[0xA] = false
			case glfw.KeyX:
				c8.Key[0x0] = false
			case glfw.KeyC:
				c8.Key[0xB] = false
			case glfw.KeyV:
				c8.Key[0xF] = false
			}
		}
	}
}

func init() {
	runtime.LockOSThread()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <rom file>\n", os.Args[0])
		os.Exit(1)
	}
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()
	width := chip8.DisplayWidth * renderScale
	height := chip8.DisplayHeight * renderScale
	window, err := glfw.CreateWindow(width, height, "Chip-8", nil, nil)
	if err != nil {
		panic(err)
	}
	c8 := chip8.New()
	if err := c8.LoadRom(os.Args[1]); err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	gl.Init()
	window.SetKeyCallback(keyHandler(c8))
	window.SetSizeCallback(resizeHandler)
	resizeHandler(window, width, height)
	for !window.ShouldClose() {
		if err := c8.Cycle(glfw.WaitEvents); err != nil {
			panic(err)
		}
		if c8.Draw {
			gl.Clear(gl.COLOR_BUFFER_BIT)
			render(c8)
			window.SwapBuffers()
		}
		glfw.PollEvents()
	}
}
