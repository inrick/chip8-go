package main

import (
	"chip8-go/chip8"
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"os"
	"runtime"
)

const (
	renderScale = 15
	// Room to draw all squares, which each require two triangles.
	vertices = chip8.DisplayWidth * chip8.DisplayHeight * 2 * 2 * 3
)

func resizeHandler(w *glfw.Window, width, height int) {
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

func fillbuf(c8 *chip8.Chip8, buf []float32) int32 {
	n := int32(0)
	for i := range c8.Gfx {
		for j := range c8.Gfx[i] {
			if c8.Gfx[i][j] == 1 {
				x := float32(i)
				y := float32(j)
				buf[n+0] = x
				buf[n+1] = y
				buf[n+2] = x + 1
				buf[n+3] = y + 1
				buf[n+4] = x + 1
				buf[n+5] = y
				buf[n+6] = x
				buf[n+7] = y
				buf[n+8] = x + 1
				buf[n+9] = y + 1
				buf[n+10] = x
				buf[n+11] = y + 1
				n += 12
			}
		}
	}
	return n / 2 // Number of vertices
}

var (
	vertexShaderGlsl = fmt.Sprintf(`
	  #version 410 core
	  in vec2 pos;
	  void main() {
	   gl_Position = vec4(pos.x/%d - 1.0, 1.0 - pos.y/%d, 0, 1.0);
	  }`, chip8.DisplayWidth/2, chip8.DisplayHeight/2)
	fragmentShaderGlsl = `
	  #version 410 core
	  out vec4 color;
	  void main() {
	    color = vec4(0.85, 0.85, 0.85, 1.0);
	  }`
)

func glSetup(buf []float32) (vao, vbo uint32, err error) {
	if err := gl.Init(); err != nil {
		return 0, 0, err
	}

	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(buf)*4, gl.Ptr(buf), gl.DYNAMIC_DRAW)

	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	cStrVshadeGlsl, freeVertexStr := gl.Strs(vertexShaderGlsl)
	gl.ShaderSource(vertexShader, 1, cStrVshadeGlsl, nil)
	gl.CompileShader(vertexShader)
	freeVertexStr()

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	cStrFshadeGlsl, freeFragmentStr := gl.Strs(fragmentShaderGlsl)
	gl.ShaderSource(fragmentShader, 1, cStrFshadeGlsl, nil)
	gl.CompileShader(fragmentShader)
	freeFragmentStr()

	// TODO log shader compilation errors

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.BindFragDataLocation(program, 0, gl.Str("color\x00"))
	gl.LinkProgram(program)
	gl.UseProgram(program)

	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, gl.PtrOffset(0))
	return
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

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

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

	buf := make([]float32, vertices, vertices)
	_, _, err = glSetup(buf)
	if err != nil {
		panic(err)
	}

	window.SetKeyCallback(keyHandler(c8))
	window.SetSizeCallback(resizeHandler)

	gl.ClearColor(.1, .1, .1, 0)
	for !window.ShouldClose() {
		if err := c8.Cycle(glfw.WaitEvents); err != nil {
			panic(err)
		}
		if c8.Draw {
			gl.Clear(gl.COLOR_BUFFER_BIT)
			n := fillbuf(c8, buf)
			// TODO this shouldn't be needed?
			gl.BufferData(gl.ARRAY_BUFFER, len(buf)*4, gl.Ptr(buf), gl.DYNAMIC_DRAW)
			gl.DrawArrays(gl.TRIANGLES, 0, n)
			window.SwapBuffers()
		}
		glfw.PollEvents()
	}
}
