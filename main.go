package main

import (
	"errors"
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/inrick/chip8-go/chip8"
	"os"
	"runtime"
	"strings"
)

const renderScale = 15

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

func fillVerticesToDraw(c8 *chip8.Chip8, vertex []uint32) int {
	h := chip8.DisplayHeight + 1
	n := 0
	for x := range c8.Gfx {
		for y := range c8.Gfx[x] {
			if c8.Gfx[x][y] == 1 {
				// Corners of quad
				q1 := uint32(x*h + y)
				q2 := uint32(x*h + y + 1)
				q3 := uint32((x+1)*h + y)
				q4 := uint32((x+1)*h + y + 1)
				vertex[n+0] = q1
				vertex[n+1] = q2
				vertex[n+2] = q3
				vertex[n+3] = q2
				vertex[n+4] = q3
				vertex[n+5] = q4
				n += 6
			}
		}
	}
	return n // Number of vertices
}

var (
	vertexShaderGlsl = `
	  #version 410 core
	  in vec2 pos;
	  void main() {
	   gl_Position = vec4(pos, 0.0, 1.0);
	  }`
	fragmentShaderGlsl = `
	  #version 410 core
	  out vec4 color;
	  void main() {
	    color = vec4(0.85, 0.85, 0.85, 1.0);
	  }`
)

func checkShaderError(shader uint32) error {
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var length int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &length)
		log := strings.Repeat("\x00", 1+int(length))
		gl.GetShaderInfoLog(shader, length, nil, gl.Str(log))
		return errors.New(log)
	}
	return nil
}

func glSetup() (vertex []uint32, vao, vbo, ebo uint32, err error) {
	if err := gl.Init(); err != nil {
		return nil, 0, 0, 0, err
	}

	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// Generate quad vertices.
	//
	// See the display pictured below. The vertices are numbered starting
	// from the top left and going down, proceeding right after the last row is
	// reached. The vertex at position (x,y) is numbered 33*x+y:
	//   - (0,0) is vertex 0
	//   - (0,1) is vertex 1
	//   - (1,0) is vertex 33
	//   - etc.
	//
	// The numbering is chosen to match the layout of chip8.Chip8.Gfx.
	//
	//      x  0 1     ...      64
	//      --->
	//  y |
	//    |  +---------------------+
	//  0 v  | . . . . . . . . . . |
	//  1    | . . . . . . . . . . |
	// ...   | . . . . . . . . . . |
	// 32    | . . . . . . . . . . |
	//       +---------------------+
	w, h := chip8.DisplayWidth+1, chip8.DisplayHeight+1
	ncoords := w * h * 2 // 2 coordinates for each vertex
	buf := make([]float32, ncoords, ncoords)
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			i := 2 * (x*h + y)
			buf[i] = -1 + float32(x)/float32(chip8.DisplayWidth/2)
			buf[i+1] = 1 - float32(y)/float32(chip8.DisplayHeight/2)
		}
	}

	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(buf)*4, gl.Ptr(buf), gl.STATIC_DRAW)

	// 65*33 quads, each quad needs 6 vertices
	vertex = make([]uint32, ncoords*3, ncoords*3)

	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(
		gl.ELEMENT_ARRAY_BUFFER, len(vertex)*4, gl.Ptr(vertex), gl.DYNAMIC_DRAW)

	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	cStrVshadeGlsl, freeVertexStr := gl.Strs(vertexShaderGlsl)
	defer freeVertexStr()
	gl.ShaderSource(vertexShader, 1, cStrVshadeGlsl, nil)
	gl.CompileShader(vertexShader)

	if err := checkShaderError(vertexShader); err != nil {
		return nil, vao, vbo, ebo, fmt.Errorf("Vertex shader error: %v", err)
	}

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	cStrFshadeGlsl, freeFragmentStr := gl.Strs(fragmentShaderGlsl)
	defer freeFragmentStr()
	gl.ShaderSource(fragmentShader, 1, cStrFshadeGlsl, nil)
	gl.CompileShader(fragmentShader)

	if err := checkShaderError(fragmentShader); err != nil {
		return nil, vao, vbo, ebo, fmt.Errorf("Fragment shader error: %v", err)
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.BindFragDataLocation(program, 0, gl.Str("color\x00"))
	gl.LinkProgram(program)
	gl.UseProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var length int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &length)
		log := strings.Repeat("\x00", 1+int(length))
		gl.GetProgramInfoLog(program, length, nil, gl.Str(log))
		return nil, vao, vbo, ebo, fmt.Errorf("Program link error: %s", log)
	}

	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, gl.PtrOffset(0))

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)

	if err := gl.GetError(); err != gl.NO_ERROR {
		return nil, vao, vbo, ebo, fmt.Errorf("GL error: 0x%x", err)
	}

	return vertex, vao, vbo, ebo, nil
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

	vertex, _, _, _, err := glSetup()
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
			n := fillVerticesToDraw(c8, vertex)
			gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, n*4, gl.Ptr(vertex))
			gl.DrawElements(gl.TRIANGLES, int32(n), gl.UNSIGNED_INT, gl.PtrOffset(0))
			window.SwapBuffers()
		}
		glfw.PollEvents()
	}
}
