/*
 * Copyright 2022 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package emu

import (
    `fmt`
    `math/bits`
    `runtime`
    `sync`
    `unsafe`

    `github.com/cloudwego/frugal/internal/atm/ir`
)

type Value struct {
    U uint64
    P unsafe.Pointer
}

type Emulator struct {
    pc *ir.Ir
    uv [6]uint64
    pv [7]unsafe.Pointer
    ar [8]Value
    rv [8]Value
}

var (
    emulatorPool sync.Pool
)

func LoadProgram(p ir.Program) (e *Emulator) {
    if v := emulatorPool.Get(); v == nil {
        return &Emulator{pc: p.Head}
    } else {
        return v.(*Emulator).Reset(p)
    }
}

func (self *Emulator) trap() {
    println("****** DEBUGGER BREAK ******")
    println("Current State:", self.String())
    runtime.Breakpoint()
}
func (self *Emulator) Ru(i int) uint64         { return self.rv[i].U }
func (self *Emulator) Rp(i int) unsafe.Pointer { return self.rv[i].P }

func (self *Emulator) Au(i int, v uint64)         *Emulator { self.ar[i].U = v; return self }
func (self *Emulator) Ap(i int, v unsafe.Pointer) *Emulator { self.ar[i].P = v; return self }

func (self *Emulator) Run() {
    var v uint64
    var p *ir.Ir
    var q *ir.Ir

    /* run until end */
    for self.pc != nil {
        p, self.pc = self.pc, self.pc.Ln
        self.uv[ir.Rz], self.pv[ir.Pn] = 0, nil

        /* main switch on OpCode */
        switch p.Op {
            default          : return
            case ir.OP_nop   : break
            case ir.OP_ip    : self.pv[p.Pd] = p.Pr
            case ir.OP_lb    : self.uv[p.Rx] = uint64(*(*int8)(unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(p.Iv))))
            case ir.OP_lw    : self.uv[p.Rx] = uint64(*(*int16)(unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(p.Iv))))
            case ir.OP_ll    : self.uv[p.Rx] = uint64(*(*int32)(unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(p.Iv))))
            case ir.OP_lq    : self.uv[p.Rx] = uint64(*(*int64)(unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(p.Iv))))
            case ir.OP_lp    : self.pv[p.Pd] = *(*unsafe.Pointer)(unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(p.Iv)))
            case ir.OP_sb    : *(*int8)(unsafe.Pointer(uintptr(self.pv[p.Pd]) + uintptr(p.Iv))) = int8(self.uv[p.Rx])
            case ir.OP_sw    : *(*int16)(unsafe.Pointer(uintptr(self.pv[p.Pd]) + uintptr(p.Iv))) = int16(self.uv[p.Rx])
            case ir.OP_sl    : *(*int32)(unsafe.Pointer(uintptr(self.pv[p.Pd]) + uintptr(p.Iv))) = int32(self.uv[p.Rx])
            case ir.OP_sq    : *(*int64)(unsafe.Pointer(uintptr(self.pv[p.Pd]) + uintptr(p.Iv))) = int64(self.uv[p.Rx])
            case ir.OP_sp    : *(*unsafe.Pointer)(unsafe.Pointer(uintptr(self.pv[p.Pd]) + uintptr(p.Iv))) = self.pv[p.Ps]
            case ir.OP_ldaq  : self.uv[p.Rx] = self.ar[p.Iv].U
            case ir.OP_ldap  : self.pv[p.Pd] = self.ar[p.Iv].P
            case ir.OP_strq  : self.rv[p.Iv].U = self.uv[p.Rx]
            case ir.OP_strp  : self.rv[p.Iv].P = self.pv[p.Ps]
            case ir.OP_addp  : self.pv[p.Pd] = unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(self.uv[p.Rx]))
            case ir.OP_subp  : self.pv[p.Pd] = unsafe.Pointer(uintptr(self.pv[p.Ps]) - uintptr(self.uv[p.Rx]))
            case ir.OP_addpi : self.pv[p.Pd] = unsafe.Pointer(uintptr(self.pv[p.Ps]) + uintptr(p.Iv))
            case ir.OP_add   : self.uv[p.Rz] = self.uv[p.Rx] + self.uv[p.Ry]
            case ir.OP_sub   : self.uv[p.Rz] = self.uv[p.Rx] - self.uv[p.Ry]
            case ir.OP_addi  : self.uv[p.Ry] = self.uv[p.Rx] + uint64(p.Iv)
            case ir.OP_muli  : self.uv[p.Ry] = self.uv[p.Rx] * uint64(p.Iv)
            case ir.OP_andi  : self.uv[p.Ry] = self.uv[p.Rx] & uint64(p.Iv)
            case ir.OP_xori  : self.uv[p.Ry] = self.uv[p.Rx] ^ uint64(p.Iv)
            case ir.OP_shri  : self.uv[p.Ry] = self.uv[p.Rx] >> p.Iv
            case ir.OP_sbiti : self.uv[p.Ry] = self.uv[p.Rx] | (1 << p.Iv)
            case ir.OP_swapw : self.uv[p.Ry] = uint64(bits.ReverseBytes16(uint16(self.uv[p.Rx])))
            case ir.OP_swapl : self.uv[p.Ry] = uint64(bits.ReverseBytes32(uint32(self.uv[p.Rx])))
            case ir.OP_swapq : self.uv[p.Ry] = bits.ReverseBytes64(self.uv[p.Rx])
            case ir.OP_beq   : if       self.uv[p.Rx]  ==       self.uv[p.Ry]  { self.pc = p.Br }
            case ir.OP_bne   : if       self.uv[p.Rx]  !=       self.uv[p.Ry]  { self.pc = p.Br }
            case ir.OP_blt   : if int64(self.uv[p.Rx]) <  int64(self.uv[p.Ry]) { self.pc = p.Br }
            case ir.OP_bltu  : if       self.uv[p.Rx]  <        self.uv[p.Ry]  { self.pc = p.Br }
            case ir.OP_bgeu  : if       self.uv[p.Rx]  >=       self.uv[p.Ry]  { self.pc = p.Br }
            case ir.OP_beqn  : if       self.pv[p.Ps]  ==                 nil  { self.pc = p.Br }
            case ir.OP_bnen  : if       self.pv[p.Ps]  !=                 nil  { self.pc = p.Br }
            case ir.OP_jal   : self.pv[p.Pd], self.pc = unsafe.Pointer(self.pc), p.Br
            case ir.OP_bzero : memclrNoHeapPointers(self.pv[p.Pd], uintptr(p.Iv))
            case ir.OP_bcopy : memmove(self.pv[p.Pd], self.pv[p.Ps], uintptr(self.uv[p.Rx]))
            case ir.OP_halt  : self.pc = nil
            case ir.OP_break : self.trap()

            /* call to C / Go / Go interface functions */
            case ir.OP_ccall: fallthrough
            case ir.OP_gcall: fallthrough
            case ir.OP_icall: ir.LookupCall(p.Iv).Call(self, p)

            /* bit test and set */
            case ir.OP_bts: {
                x := self.uv[p.Rx]
                y := self.uv[p.Ry]

                /* test and set the bit */
                if self.uv[p.Ry] |= 1 << x; y & (1 << x) == 0 {
                    self.uv[p.Rz] = 0
                } else {
                    self.uv[p.Rz] = 1
                }
            }

            /* table switch */
            case ir.OP_bsw: {
                if v = self.uv[p.Rx]; v < uint64(p.Iv) {
                    if q = *(**ir.Ir)(unsafe.Pointer(uintptr(p.Pr) + uintptr(v) * 8)); q != nil {
                        self.pc = q
                    }
                }
            }
        }
    }

    /* check for exceptions */
    if self.pc != nil {
        panic(fmt.Sprintf("illegal OpCode: %#02x", self.pc.Op))
    }
}


func (self *Emulator) Free() {
    emulatorPool.Put(self)
}

func (self *Emulator) Reset(p ir.Program) *Emulator {
    *self = Emulator{pc: p.Head}
    return self
}

/** Implementation of ir.CallState **/

func (self *Emulator) Gr(id ir.GenericRegister) uint64 {
    return self.uv[id]
}

func (self *Emulator) Pr(id ir.PointerRegister) unsafe.Pointer {
    return self.pv[id]
}

func (self *Emulator) SetGr(id ir.GenericRegister, val uint64) {
    self.uv[id] = val
}

func (self *Emulator) SetPr(id ir.PointerRegister, val unsafe.Pointer) {
    self.pv[id] = val
}

/** State Dumping **/

const _F_emulator = `Emulator {
    pc  (%p)%s
    r0  %#x
    r1  %#x
    r2  %#x
    r3  %#x
    r4  %#x
    r5  %#x
   ----
    p0  %p
    p1  %p
    p2  %p
    p3  %p
    p4  %p
    p5  %p
    p6  %p
}`

func (self *Emulator) String() string {
    return fmt.Sprintf(
        _F_emulator,
        self.pc,
        self.pc.Disassemble(nil),
        self.uv[0],
        self.uv[1],
        self.uv[2],
        self.uv[3],
        self.uv[4],
        self.uv[5],
        self.pv[0],
        self.pv[1],
        self.pv[2],
        self.pv[3],
        self.pv[4],
        self.pv[5],
        self.pv[6],
    )
}
