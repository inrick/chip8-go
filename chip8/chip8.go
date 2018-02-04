// Package chip8 implements a Chip-8 interpreter.
// Follows description in Cowgod's Chip-8 Technical Reference v1.0 [1] and
// How to write an emulator [2].
//
//   [1] http://devernay.free.fr/hacks/chip8/C8TECH10.HTM
//   [2] http://www.multigesture.net/articles/how-to-write-an-emulator-chip-8-interpreter/
package chip8

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
)

const (
	DisplayWidth  = 64
	DisplayHeight = 32
	maxRomSize    = 0xfff - 0x200 + 1
)

var fontset = [...]uint8{
	0xf0, 0x90, 0x90, 0x90, 0xf0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xf0, 0x10, 0xf0, 0x80, 0xf0, // 2
	0xf0, 0x10, 0xf0, 0x10, 0xf0, // 3
	0x90, 0x90, 0xf0, 0x10, 0x10, // 4
	0xf0, 0x80, 0xf0, 0x10, 0xf0, // 5
	0xf0, 0x80, 0xf0, 0x90, 0xf0, // 6
	0xf0, 0x10, 0x20, 0x40, 0x40, // 7
	0xf0, 0x90, 0xf0, 0x90, 0xf0, // 8
	0xf0, 0x90, 0xf0, 0x10, 0xf0, // 9
	0xf0, 0x90, 0xf0, 0x90, 0x90, // A
	0xe0, 0x90, 0xe0, 0x90, 0xe0, // B
	0xf0, 0x80, 0x80, 0x80, 0xf0, // C
	0xe0, 0x90, 0x90, 0x90, 0xe0, // D
	0xf0, 0x80, 0xf0, 0x80, 0xf0, // E
	0xf0, 0x80, 0xf0, 0x80, 0x80, // F
}

type opcode uint16

type Chip8 struct {
	Gfx    [DisplayWidth][DisplayHeight]uint8
	Key    [0x10]bool
	Draw   bool
	mem    [0x1000]uint8
	v      [0x10]uint8
	stack  [0x10]uint16
	i, pc  uint16
	sp     uint8
	dt, st uint8 // Delay timer & sound timer
}

func New() *Chip8 {
	c8 := new(Chip8)
	for i, x := range fontset {
		c8.mem[i] = x
	}
	c8.pc = 0x200
	return c8
}

func (c8 *Chip8) LoadRom(romPath string) error {
	rom, err := os.Open(romPath)
	if err != nil {
		return fmt.Errorf("Error reading ROM file: %v", err)
	}
	defer rom.Close()
	bytesRead, err := rom.Read(c8.mem[0x200:])
	if err != nil {
		return fmt.Errorf("Error reading ROM file: %v", err)
	}
	if bytesRead > maxRomSize {
		return errors.New("ROM file too big")
	}
	return nil
}

func (c8 *Chip8) incPc(skipNextInstruction bool) {
	if skipNextInstruction {
		c8.pc += 4
	} else {
		c8.pc += 2
	}
}

// Emulates one Chip-8 cycle. Comments describing opcodes are copied from
// Cowgod's reference [1].
func (c8 *Chip8) Cycle(waitForInput func()) error {
	op := (uint16(c8.mem[c8.pc]) << 8) | uint16(c8.mem[c8.pc+1])
	c8.Draw = false
	switch op & 0xf000 {
	case 0x0000:
		switch op & 0xff {
		case 0xe0:
			// 00E0 - CLS -- Clear the display.
			for i := range c8.Gfx {
				for j := range c8.Gfx[i] {
					c8.Gfx[i][j] = 0
				}
			}
			c8.Draw = true
		case 0xee:
			// 00EE - RET -- Return from a subroutine.
			c8.sp--
			c8.pc = c8.stack[c8.sp]
		default:
			// 0nnn - SYS addr -- Jump to a machine code routine at nnn.
			// Apparently ignored in modern interpreters.
			goto Unknown
		}
		c8.incPc(false)
	case 0x1000:
		// 1nnn - JP addr -- Jump to location nnn.
		c8.pc = op & 0xfff
	case 0x2000:
		// 2nnn - CALL addr -- Call subroutine at nnn.
		c8.stack[c8.sp] = c8.pc
		c8.sp++
		c8.pc = op & 0xfff
	case 0x3000:
		// 3xkk - SE Vx, byte -- Skip next instruction if Vx = kk.
		x := uint8((op & 0xf00) >> 8)
		kk := uint8(op & 0xff)
		c8.incPc(c8.v[x] == kk)
	case 0x4000:
		// 4xkk - SNE Vx, byte -- Skip next instruction if Vx != kk.
		x := uint8((op & 0xf00) >> 8)
		kk := uint8(op & 0xff)
		c8.incPc(c8.v[x] != kk)
	case 0x5000:
		switch op & 0xf {
		case 0x0:
			// 5xy0 - SE Vx, Vy -- Skip next instruction if Vx = Vy.
			x := uint8((op & 0xf00) >> 8)
			y := uint8((op & 0xf0) >> 4)
			c8.incPc(c8.v[x] == c8.v[y])
		default:
			goto Unknown
		}
	case 0x6000:
		// 6xkk - LD Vx, byte -- Set Vx = kk.
		x := uint8((op & 0xf00) >> 8)
		kk := uint8(op & 0xff)
		c8.v[x] = kk
		c8.incPc(false)
	case 0x7000:
		// 7xkk - ADD Vx, byte -- Set Vx = Vx + kk.
		x := uint8((op & 0xf00) >> 8)
		kk := uint8(op & 0xff)
		c8.v[x] += kk
		c8.incPc(false)
	case 0x8000:
		// 8XYN X and Y identify data registers, N the operation
		x := uint8((op & 0xf00) >> 8)
		y := uint8((op & 0xf0) >> 4)
		switch op & 0xf {
		case 0x0:
			// 8xy0 - LD Vx, Vy -- Set Vx = Vy.
			c8.v[x] = c8.v[y]
		case 0x1:
			// 8xy1 - OR Vx, Vy -- Set Vx = Vx OR Vy.
			c8.v[x] |= c8.v[y]
		case 0x2:
			// 8xy2 - AND Vx, Vy -- Set Vx = Vx AND Vy.
			c8.v[x] &= c8.v[y]
		case 0x3:
			// 8xy3 - XOR Vx, Vy -- Set Vx = Vx XOR Vy.
			c8.v[x] ^= c8.v[y]
		case 0x4:
			// 8xy4 - ADD Vx, Vy -- Set Vx = Vx + Vy, set VF = carry.
			if c8.v[y] > (0xff - c8.v[x]) {
				c8.v[0xf] = 1
			} else {
				c8.v[0xf] = 0
			}
			c8.v[x] += c8.v[y]
		case 0x5:
			// 8xy5 - SUB Vx, Vy -- Set Vx = Vx - Vy, set VF = NOT borrow.
			if c8.v[x] > c8.v[y] {
				c8.v[0xf] = 1
			} else {
				c8.v[0xf] = 0
			}
			c8.v[x] -= c8.v[y]
		case 0x6:
			// 8xy6 - SHR Vx {, Vy} -- Set Vx = Vx SHR 1.
			c8.v[0xf] = c8.v[x] & 0x1
			c8.v[x] >>= 1
		case 0x7:
			// 8xy7 - SUBN Vx, Vy -- Set Vx = Vy - Vx, set VF = NOT borrow.
			x := uint8((op & 0xf00) >> 8)
			y := uint8((op & 0xf0) >> 4)
			if c8.v[y] > c8.v[x] {
				c8.v[0xf] = 1
			} else {
				c8.v[0xf] = 0
			}
			c8.v[x] = c8.v[y] - c8.v[x]
		case 0xe:
			// 8xyE - SHL Vx {, Vy} -- Set Vx = Vx SHL 1.
			x := uint8((op & 0xf00) >> 8)
			c8.v[0xf] = (x & 0x80) >> 7
			c8.v[x] <<= 1
		default:
			goto Unknown
		}
		c8.incPc(false)
	case 0x9000:
		switch op & 0xf {
		case 0x0:
			// 9xy0 - SNE Vx, Vy -- Skip next instruction if Vx != Vy.
			x := uint8((op & 0xf00) >> 8)
			y := uint8((op & 0xf0) >> 4)
			c8.incPc(c8.v[x] != c8.v[y])
		default:
			goto Unknown
		}
	case 0xa000:
		// Annn - LD I, addr -- Set I = nnn.
		c8.i = op & 0xfff
		c8.incPc(false)
	case 0xb000:
		// Bnnn - JP V0, addr -- Jump to location nnn + V0.
		c8.pc = (op & 0xfff) + uint16(c8.v[0])
	case 0xc000:
		// Cxkk - RND Vx, byte -- Set Vx = random byte AND kk.
		x := uint8((op & 0xf00) >> 8)
		kk := uint8(op & 0xff)
		c8.v[x] = kk & uint8(rand.Intn(0x100))
		c8.incPc(false)
	case 0xd000:
		// Dxyn - DRW Vx, Vy, nibble -- Display n-byte sprite starting at memory
		// location I at (Vx, Vy), set VF = collision.
		x := uint8((op & 0xf00) >> 8)
		y := uint8((op & 0xf0) >> 4)
		n := uint8(op & 0xf)
		c8.v[0xf] = 0
		for row := uint8(0); row < n; row++ {
			spriteRow := c8.mem[c8.i+uint16(row)]
			for col := uint8(0); col < 8; col++ {
				if spriteRow&uint8(0x1<<(7-col)) != 0 {
					// Wrap around if sprite is at the edge
					i := (c8.v[x] + col) % DisplayWidth
					j := (c8.v[y] + row) % DisplayHeight
					c8.Gfx[i][j] ^= 1
					if c8.Gfx[i][j] == 0 {
						c8.v[0xf] = 1
					}
				}
			}
		}
		c8.Draw = true
		c8.incPc(false)
	case 0xe000:
		x := uint8((op & 0xf00) >> 8)
		switch op & 0xff {
		case 0x9e:
			// Ex9E - SKP Vx -- Skip next instruction if key with the value of Vx is
			// pressed.
			c8.incPc(c8.Key[c8.v[x]])
		case 0xa1:
			// ExA1 - SKNP Vx -- Skip next instruction if key with the value of Vx is
			// not pressed.
			c8.incPc(!c8.Key[c8.v[x]])
		default:
			goto Unknown
		}
	case 0xf000:
		x := uint8((op & 0xf00) >> 8)
		switch op & 0xff {
		case 0x7:
			// Fx07 - LD Vx, DT -- Set Vx = delay timer value.
			c8.v[x] = c8.dt
		case 0xa:
			// Fx0A - LD Vx, K -- Wait for a key press, store the value of the key in
			// Vx.
		Waiting:
			for {
				waitForInput()
				for i := uint8(0); i < 0x10; i++ {
					if c8.Key[i] {
						c8.v[x] = i
						break Waiting
					}
				}
			}
		case 0x15:
			// Fx15 - LD DT, Vx -- Set delay timer = Vx.
			c8.dt = c8.v[x]
		case 0x18:
			// Fx18 - LD ST, Vx -- Set sound timer = Vx.
			c8.st = c8.v[x]
		case 0x1e:
			// Fx1E - ADD I, Vx -- Set I = I + Vx.
			c8.i += uint16(c8.v[x])
		case 0x29:
			// Fx29 - LD F, Vx -- Set I = location of sprite for digit Vx.
			if c8.v[x] > 0xf {
				return fmt.Errorf("Expected Vx <= 0xf but found Vx=0x%x", c8.v[x])
			}
			c8.i = uint16(c8.v[x]) * 5
		case 0x33:
			// Fx33 - LD B, Vx -- Store BCD representation of Vx in memory locations
			// I, I+1, and I+2.
			c8.mem[c8.i] = c8.v[x] / 100
			c8.mem[c8.i+1] = (c8.v[x] % 100) / 10
			c8.mem[c8.i+2] = c8.v[x] % 10
		case 0x55:
			// Fx55 - LD [I], Vx -- Store registers V0 through Vx in memory starting
			// at location I.
			for i := uint8(0); i < x+1; i++ {
				c8.mem[c8.i+uint16(i)] = c8.v[i]
			}
		case 0x65:
			// Fx65 - LD Vx, [I] -- Read registers V0 through Vx from memory starting
			// at location I.
			for i := uint8(0); i < x+1; i++ {
				c8.v[i] = c8.mem[c8.i+uint16(i)]
			}
		default:
			goto Unknown
		}
		c8.incPc(false)
	default:
		goto Unknown
	}
	// TODO timers should be decremented at 60 hz rate
	if c8.dt > 0 {
		c8.dt--
	}
	if c8.st > 0 {
		fmt.Print("\a")
		c8.st--
	}
	return nil
Unknown:
	return fmt.Errorf("Unknown opcode 0x%x", op)
}
