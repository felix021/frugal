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

package decoder

import (
    `fmt`

    `github.com/cloudwego/frugal/internal/atm/ir`
    `github.com/cloudwego/frugal/internal/binary/defs`
    `github.com/cloudwego/frugal/internal/rt`
)

/** Function Prototype
 *
 *      func(buf unsafe.Pointer, nb int, i int, p unsafe.Pointer, rs *RuntimeState, st int) (pos int, err error)
 */

const (
    ARG_buf = 0
    ARG_nb  = 1
    ARG_i   = 2
    ARG_p   = 3
    ARG_rs  = 4
    ARG_st  = 5
)

const (
    RET_pos      = 0
    RET_err_itab = 1
    RET_err_data = 2
)

/** Register Allocations
 *
 *      P1      Current Working Pointer
 *      P2      Input Buffer Pointer
 *      P3      Runtime State Pointer
 *      P4      Error Type Pointer
 *      P5      Error Value Pointer
 *
 *      R2      Input Cursor
 *      R3      State Index
 *      R4      Field Tag
 */

const (
    WP = ir.P1
    IP = ir.P2
    RS = ir.P3
    ET = ir.P4      // may also be used as a temporary pointer register
    EP = ir.P5      // may also be used as a temporary pointer register
)

const (
    IC = ir.R2
    ST = ir.R3
    TG = ir.R4
)

const (
    TP = ir.P0
    TR = ir.R0
    UR = ir.R1
)

const (
    LB_eof      = "_eof"
    LB_halt     = "_halt"
    LB_type     = "_type"
    LB_skip     = "_skip"
    LB_error    = "_error"
    LB_missing  = "_missing"
    LB_overflow = "_overflow"
)

var (
    _E_overflow  error
    _V_zerovalue uint64
)

func init() {
    _E_overflow = fmt.Errorf("frugal: decoder stack overflow")
}

func Translate(s Program) ir.Program {
    p := ir.CreateBuilder()
    prologue (p)
    program  (p, s)
    epilogue (p)
    errors   (p)
    return p.Build()
}

func errors(p *ir.Builder) {
    p.Label (LB_eof)
    p.LDAQ  (ARG_nb, UR)
    p.SUB   (TR, UR, TR)
    p.GCALL (F_error_eof).
      A0    (TR).
      R0    (ET).
      R1    (EP)
    p.JAL   (LB_error, ir.Pn)
    p.Label (LB_type)
    p.GCALL (F_error_type).
      A0    (UR).
      A1    (TR).
      R0    (ET).
      R1    (EP)
    p.JAL   (LB_error, ir.Pn)
    p.Label (LB_skip)
    p.GCALL (F_error_skip).
      A0    (TR).
      R0    (ET).
      R1    (EP)
    p.JAL   (LB_error, ir.Pn)
    p.Label (LB_missing)
    p.GCALL (F_error_missing).
      A0    (ET).
      A1    (UR).
      A2    (TR).
      R0    (ET).
      R1    (EP)
    p.JAL   (LB_error, ir.Pn)
    p.Label (LB_overflow)
    p.IP    (&_E_overflow, TP)
    p.LP    (TP, 0, ET)
    p.LP    (TP, 8, EP)
    p.JAL   (LB_error, ir.Pn)
}

func program(p *ir.Builder, s Program) {
    for i, v := range s {
        p.Mark(i)
        translators[v.Op](p, v)
    }
}

func prologue(p *ir.Builder) {
    p.LDAP  (ARG_buf, IP)
    p.LDAQ  (ARG_i, IC)
    p.LDAP  (ARG_p, WP)
    p.LDAP  (ARG_rs, RS)
    p.LDAQ  (ARG_st, ST)
}

func epilogue(p *ir.Builder) {
    p.Label (LB_halt)
    p.MOVP  (ir.Pn, ET)
    p.MOVP  (ir.Pn, EP)
    p.Label (LB_error)
    p.STRQ  (IC, RET_pos)
    p.STRP  (ET, RET_err_itab)
    p.STRP  (EP, RET_err_data)
    p.HALT  ()
}

var translators = [256]func(*ir.Builder, Instr) {
    OP_int               : translate_OP_int,
    OP_str               : translate_OP_str,
    OP_bin               : translate_OP_bin,
    OP_enum              : translate_OP_enum,
    OP_size              : translate_OP_size,
    OP_type              : translate_OP_type,
    OP_seek              : translate_OP_seek,
    OP_deref             : translate_OP_deref,
    OP_ctr_load          : translate_OP_ctr_load,
    OP_ctr_decr          : translate_OP_ctr_decr,
    OP_ctr_is_zero       : translate_OP_ctr_is_zero,
    OP_map_alloc         : translate_OP_map_alloc,
    OP_map_close         : translate_OP_map_close,
    OP_map_set_i8        : translate_OP_map_set_i8,
    OP_map_set_i16       : translate_OP_map_set_i16,
    OP_map_set_i32       : translate_OP_map_set_i32,
    OP_map_set_i64       : translate_OP_map_set_i64,
    OP_map_set_str       : translate_OP_map_set_str,
    OP_map_set_pointer   : translate_OP_map_set_pointer,
    OP_list_alloc        : translate_OP_list_alloc,
    OP_struct_skip       : translate_OP_struct_skip,
    OP_struct_ignore     : translate_OP_struct_ignore,
    OP_struct_bitmap     : translate_OP_struct_bitmap,
    OP_struct_switch     : translate_OP_struct_switch,
    OP_struct_require    : translate_OP_struct_require,
    OP_struct_is_stop    : translate_OP_struct_is_stop,
    OP_struct_mark_tag   : translate_OP_struct_mark_tag,
    OP_struct_read_type  : translate_OP_struct_read_type,
    OP_struct_check_type : translate_OP_struct_check_type,
    OP_make_state        : translate_OP_make_state,
    OP_drop_state        : translate_OP_drop_state,
    OP_construct         : translate_OP_construct,
    OP_defer             : translate_OP_defer,
    OP_goto              : translate_OP_goto,
    OP_halt              : translate_OP_halt,
}

func translate_OP_int(p *ir.Builder, v Instr) {
    switch v.Iv {
        case 1  : p.ADDP(IP, IC, EP); p.LB(EP, 0, TR);                  p.SB(TR, WP, 0); p.ADDI(IC, 1, IC)
        case 2  : p.ADDP(IP, IC, EP); p.LW(EP, 0, TR); p.SWAPW(TR, TR); p.SW(TR, WP, 0); p.ADDI(IC, 2, IC)
        case 4  : p.ADDP(IP, IC, EP); p.LL(EP, 0, TR); p.SWAPL(TR, TR); p.SL(TR, WP, 0); p.ADDI(IC, 4, IC)
        case 8  : p.ADDP(IP, IC, EP); p.LQ(EP, 0, TR); p.SWAPQ(TR, TR); p.SQ(TR, WP, 0); p.ADDI(IC, 8, IC)
        default : panic("can only convert 1, 2, 4 or 8 bytes at a time")
    }
}

func translate_OP_str(p *ir.Builder, _ Instr) {
    p.SP    (ir.Pn, WP, 0)
    translate_OP_binstr(p)
}

func translate_OP_bin(p *ir.Builder, _ Instr) {
    p.IP    (&_V_zerovalue, TP)
    p.SP    (TP, WP, 0)
    translate_OP_binstr(p)
    p.SQ    (TR, WP, 16)
}

func translate_OP_binstr(p *ir.Builder) {
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 4, IC)
    p.LL    (EP, 0, TR)
    p.SWAPL (TR, TR)
    p.LDAQ  (ARG_nb, UR)
    p.BLTU  (UR, TR, LB_eof)
    p.BEQ   (TR, ir.Rz, "_empty_{n}")
    p.ADDPI (EP, 4, EP)
    p.ADD   (IC, TR, IC)
    p.SP    (EP, WP, 0)
    p.Label ("_empty_{n}")
    p.SQ    (TR, WP, 8)
}

func translate_OP_enum(p *ir.Builder, _ Instr) {
    p.ADDP  (IP, IC, EP)
    p.LL    (EP, 0, TR)
    p.SWAPL (TR, TR)
    p.SQ    (TR, WP, 0)
    p.ADDI  (IC, 4, IC)
}

func translate_OP_size(p *ir.Builder, v Instr) {
    p.IQ    (v.Iv, TR)
    p.LDAQ  (ARG_nb, UR)
    p.BLTU  (UR, TR, LB_eof)
}

func translate_OP_type(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, TP)
    p.LB    (TP, 0, TR)
    p.IB    (int8(v.Tx), UR)
    p.BNE   (TR, UR, LB_type)
    p.ADDI  (IC, 1, IC)
}

func translate_OP_seek(p *ir.Builder, v Instr) {
    p.ADDPI (WP, v.Iv, WP)
}

func translate_OP_deref(p *ir.Builder, v Instr) {
    p.LQ    (WP, 0, TR)
    p.BNE   (TR, ir.Rz, "_skip_{n}")
    p.IB    (1, UR)
    p.IP    (v.Vt, TP)
    p.IQ    (int64(v.Vt.Size), TR)
    p.GCALL (F_mallocgc).
      A0    (TR).
      A1    (TP).
      A2    (UR).
      R0    (TP)
    p.SP    (TP, WP, 0)
    p.Label ("_skip_{n}")
    p.LP    (WP, 0, WP)
}

func translate_OP_ctr_load(p *ir.Builder, _ Instr) {
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 4, IC)
    p.LL    (EP, 0, TR)
    p.SWAPL (TR, TR)
    p.ADDP  (RS, ST, TP)
    p.SQ    (TR, TP, NbOffset)
}

func translate_OP_ctr_decr(p *ir.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)
    p.LQ    (TP, NbOffset, TR)
    p.SUBI  (TR, 1, TR)
    p.SQ    (TR, TP, NbOffset)
}

func translate_OP_ctr_is_zero(p *ir.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)
    p.LQ    (TP, NbOffset, TR)
    p.BEQ   (TR, ir.Rz, p.At(v.To))
}

func translate_OP_map_alloc(p *ir.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)
    p.LQ    (TP, NbOffset, TR)
    p.LP    (WP, 0, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_makemap).
      A0    (ET).
      A1    (TR).
      A2    (TP).
      R0    (TP)
    p.SP    (TP, WP, 0)
    p.ADDP  (RS, ST, EP)
    p.SP    (TP, EP, MpOffset)
}

func translate_OP_map_close(p *ir.Builder, _ Instr) {
    p.ADDP  (RS, ST, TP)
    p.SP    (ir.Pn, TP, MpOffset)
}

func translate_OP_map_set_i8(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, EP)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign).
      A0    (ET).
      A1    (TP).
      A2    (EP).
      R0    (WP)
    p.ADDI  (IC, 1, IC)
}

func translate_OP_map_set_i16(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, ET)
    p.ADDI  (IC, 2, IC)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, EP)
    p.LW    (ET, 0, TR)
    p.SWAPW (TR, TR)
    p.SW    (TR, RS, IvOffset)
    p.ADDPI (RS, IvOffset, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign).
      A0    (ET).
      A1    (EP).
      A2    (TP).
      R0    (WP)
}

func translate_OP_map_set_i32(p *ir.Builder, v Instr) {
    if rt.MapType(v.Vt).IsFastMap() {
        translate_OP_map_set_i32_fast(p, v)
    } else {
        translate_OP_map_set_i32_safe(p, v)
    }
}

func translate_OP_map_set_i32_fast(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 4, IC)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, TP)
    p.LL    (EP, 0, TR)
    p.SWAPL (TR, TR)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign_fast32).
      A0    (ET).
      A1    (TP).
      A2    (TR).
      R0    (WP)
}

func translate_OP_map_set_i32_safe(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, ET)
    p.ADDI  (IC, 4, IC)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, EP)
    p.LL    (ET, 0, TR)
    p.SWAPL (TR, TR)
    p.SL    (TR, RS, IvOffset)
    p.ADDPI (RS, IvOffset, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign).
      A0    (ET).
      A1    (EP).
      A2    (TP).
      R0    (WP)
}

func translate_OP_map_set_i64(p *ir.Builder, v Instr) {
    if rt.MapType(v.Vt).IsFastMap() {
        translate_OP_map_set_i64_fast(p, v)
    } else {
        translate_OP_map_set_i64_safe(p, v)
    }
}

func translate_OP_map_set_i64_fast(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 8, IC)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, TP)
    p.LQ    (EP, 0, TR)
    p.SWAPQ (TR, TR)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign_fast64).
      A0    (ET).
      A1    (TP).
      A2    (TR).
      R0    (WP)
}

func translate_OP_map_set_i64_safe(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, ET)
    p.ADDI  (IC, 2, IC)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, EP)
    p.LQ    (ET, 0, TR)
    p.SWAPQ (TR, TR)
    p.SQ    (TR, RS, IvOffset)
    p.ADDPI (RS, IvOffset, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign).
      A0    (ET).
      A1    (EP).
      A2    (TP).
      R0    (WP)
}

func translate_OP_map_set_str(p *ir.Builder, v Instr) {
    if rt.MapType(v.Vt).IsFastMap() {
        translate_OP_map_set_str_fast(p, v)
    } else {
        translate_OP_map_set_str_safe(p, v)
    }
}

func translate_OP_map_set_str_fast(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 4, IC)
    p.LL    (EP, 0, TR)
    p.SWAPL (TR, TR)
    p.LDAQ  (ARG_nb, UR)
    p.BLTU  (UR, TR, LB_eof)
    p.MOVP  (ir.Pn, EP)
    p.BEQ   (TR, ir.Rz, "_empty_{n}")
    p.ADDP  (IP, IC, EP)
    p.ADD   (IC, TR, IC)
    p.Label ("_empty_{n}")
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign_faststr).
      A0    (ET).
      A1    (TP).
      A2    (EP).
      A3    (TR).
      R0    (WP)
}

func translate_OP_map_set_str_safe(p *ir.Builder, v Instr) {
    p.ADDP  (IP, IC, ET)
    p.ADDI  (IC, 4, IC)
    p.LL    (ET, 0, TR)
    p.SWAPL (TR, TR)
    p.LDAQ  (ARG_nb, UR)
    p.BLTU  (UR, TR, LB_eof)
    p.SQ    (TR, RS, IvOffset)
    p.SP    (ir.Pn, RS, PrOffset)
    p.BEQ   (TR, ir.Rz, "_empty_{n}")
    p.ADDPI (ET, 4, ET)
    p.ADD   (IC, TR, IC)
    p.SP    (ET, RS, PrOffset)
    p.Label ("_empty_{n}")
    p.ADDP  (RS, ST, EP)
    p.LP    (EP, MpOffset, EP)
    p.IP    (v.Vt, ET)
    p.ADDPI (RS, PrOffset, TP)
    p.GCALL (F_mapassign).
      A0    (ET).
      A1    (EP).
      A2    (TP).
      R0    (WP)
    p.SP    (ir.Pn, RS, PrOffset)
}

func translate_OP_map_set_pointer(p *ir.Builder, v Instr) {
    if rt.MapType(v.Vt).IsFastMap() {
        translate_OP_map_set_pointer_fast(p, v)
    } else {
        translate_OP_map_set_pointer_safe(p, v)
    }
}

func translate_OP_map_set_pointer_fast(p *ir.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, TP)
    p.IP    (v.Vt, ET)
    p.GCALL (F_mapassign_fast64ptr).
      A0    (ET).
      A1    (TP).
      A2    (WP).
      R0    (WP)
}

func translate_OP_map_set_pointer_safe(p *ir.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, MpOffset, EP)
    p.SP    (WP, RS, PrOffset)
    p.IP    (v.Vt, ET)
    p.ADDPI (RS, PrOffset, TP)
    p.GCALL (F_mapassign).
      A0    (ET).
      A1    (EP).
      A2    (TP).
      R0    (WP)
    p.SP    (ir.Pn, RS, PrOffset)
}

func translate_OP_list_alloc(p *ir.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)
    p.LQ    (TP, NbOffset, TR)
    p.SQ    (TR, WP, 8)
    p.LQ    (WP, 16, UR)
    p.BNE   (TR, ir.Rz, "_alloc_{n}")
    p.BNE   (UR, ir.Rz, "_done_{n}")
    p.IP    (&_V_zerovalue, TP)
    p.SP    (TP, WP, 0)
    p.SQ    (ir.Rz, WP, 16)
    p.JAL   ("_done_{n}", ir.Pn)
    p.Label ("_alloc_{n}")
    p.BGEU  (UR, TR, "_done_{n}")
    p.SQ    (TR, WP, 16)
    p.IB    (1, UR)
    p.IP    (v.Vt, TP)
    p.MULI  (TR, int64(v.Vt.Size), TR)
    p.GCALL (F_mallocgc).
      A0    (TR).
      A1    (TP).
      A2    (UR).
      R0    (TP)
    p.SP    (TP, WP, 0)
    p.Label ("_done_{n}")
    p.LP    (WP, 0, WP)
}

func translate_OP_struct_skip(p *ir.Builder, _ Instr) {
    p.ADDPI (RS, SkOffset, TP)
    p.LDAQ  (ARG_nb, TR)
    p.SUB   (TR, IC, TR)
    p.ADDP  (IP, IC, EP)
    p.CCALL (C_skip).
      A0    (TP).
      A1    (EP).
      A2    (TR).
      A3    (TG).
      R0    (TR)
    p.BLT   (TR, ir.Rz, LB_skip)
    p.ADD   (IC, TR, IC)
}

func translate_OP_struct_ignore(p *ir.Builder, _ Instr) {
    p.ADDPI (RS, SkOffset, TP)
    p.LDAQ  (ARG_nb, TR)
    p.SUB   (TR, IC, TR)
    p.ADDP  (IP, IC, EP)
    p.IB    (int8(defs.T_struct), TG)
    p.CCALL (C_skip).
      A0    (TP).
      A1    (EP).
      A2    (TR).
      A3    (TG).
      R0    (TR)
    p.BLT   (TR, ir.Rz, LB_skip)
    p.ADD   (IC, TR, IC)
}

func translate_OP_struct_bitmap(p *ir.Builder, v Instr) {
    buf := newFieldBitmap()
    tab := mkstab(v.Sw, v.Iv)

    /* add all the bits */
    for _, i := range tab {
        buf.Append(i)
    }

    /* allocate a new bitmap */
    p.GCALL (F_newFiedBitmap).R0(TP)
    p.ADDP  (RS, ST, EP)
    p.SP    (TP, EP, FmOffset)

    /* clear bits of required fields if any */
    for i := int64(0); i < MaxBitmap; i++ {
        if buf[i] != 0 {
            p.SQ(ir.Rz, TP, i * 8)
        }
    }

    /* release the buffer */
    buf.Clear()
    buf.Free()
}

func translate_OP_struct_switch(p *ir.Builder, v Instr) {
    stab := mkstab(v.Sw, v.Iv)
    ptab := make([]string, v.Iv)

    /* convert the switch table */
    for i, to := range stab {
        if to >= 0 {
            ptab[i] = p.At(to)
        }
    }

    /* load and dispatch the field */
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 2, IC)
    p.LW    (EP, 0, TR)
    p.SWAPW (TR, TR)
    p.BSW   (TR, ptab)
}

func translate_OP_struct_require(p *ir.Builder, v Instr) {
    buf := newFieldBitmap()
    tab := mkstab(v.Sw, v.Iv)

    /* add all the bits */
    for _, i := range tab {
        buf.Append(i)
    }

    /* load the bitmap */
    p.ADDP  (RS, ST, EP)
    p.LP    (EP, FmOffset, TP)

    /* test mask for each word if any */
    for i := int64(0); i < MaxBitmap; i++ {
        if buf[i] != 0 {
            p.LQ    (TP, i * 8, TR)
            p.ANDI  (TR, buf[i], TR)
            p.XORI  (TR, buf[i], TR)
            p.IQ    (i, UR)
            p.IP    (v.Vt, ET)
            p.BNE   (TR, ir.Rz, LB_missing)
        }
    }

    /* free the bitmap */
    p.SP    (ir.Pn, EP, FmOffset)
    p.GCALL (F_FieldBitmap_Free).A0(TP)

    /* release the buffer */
    buf.Clear()
    buf.Free()
}

func translate_OP_struct_is_stop(p *ir.Builder, v Instr) {
    p.BEQ   (TG, ir.Rz, p.At(v.To))
}

func translate_OP_struct_mark_tag(p *ir.Builder, v Instr) {
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, FmOffset, TP)
    p.LQ    (TP, v.Iv / 64 * 8, TR)
    p.SBITI (TR, v.Iv % 64, TR)
    p.SQ    (TR, TP, v.Iv / 64 * 8)
}

func translate_OP_struct_read_type(p *ir.Builder, _ Instr) {
    p.ADDP  (IP, IC, EP)
    p.ADDI  (IC, 1, IC)
    p.LB    (EP, 0, TG)
}

func translate_OP_struct_check_type(p *ir.Builder, v Instr) {
    p.IB    (int8(v.Tx), TR)
    p.BNE   (TG, TR, p.At(v.To))
}

func translate_OP_make_state(p *ir.Builder, _ Instr) {
    p.IQ    (StateMax, TR)
    p.BGEU  (ST, TR, LB_overflow)
    p.ADDP  (RS, ST, TP)
    p.SP    (WP, TP, WpOffset)
    p.ADDI  (ST, StateSize, ST)
}

func translate_OP_drop_state(p *ir.Builder, _ Instr) {
    p.SUBI  (ST, StateSize, ST)
    p.ADDP  (RS, ST, TP)
    p.LP    (TP, WpOffset, WP)
    p.SP    (ir.Pn, TP, WpOffset)
}

func translate_OP_construct(p *ir.Builder, v Instr) {
    p.IB    (1, UR)
    p.IP    (v.Vt, TP)
    p.IQ    (int64(v.Vt.Size), TR)
    p.GCALL (F_mallocgc).
      A0    (TR).
      A1    (TP).
      A2    (UR).
      R0    (WP)
}

func translate_OP_defer(p *ir.Builder, v Instr) {
    p.IP    (v.Vt, TP)
    p.LDAQ  (ARG_nb, TR)
    p.GCALL (F_decode).
      A0    (TP).
      A1    (IP).
      A2    (TR).
      A3    (IC).
      A4    (WP).
      A5    (RS).
      A6    (ST).
      R0    (IC).
      R1    (ET).
      R2    (EP)
    p.BNEN  (ET, LB_error)
}

func translate_OP_goto(p *ir.Builder, v Instr) {
    p.JAL   (p.At(v.To), ir.Pn)
}

func translate_OP_halt(p *ir.Builder, _ Instr) {
    p.JAL   (LB_halt, ir.Pn)
}
