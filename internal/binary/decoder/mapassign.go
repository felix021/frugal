/*
 * Copyright 2021 ByteDance Inc.
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
    `unsafe`

    `github.com/cloudwego/frugal/internal/rt`
)

//go:noescape
//go:linkname mapassign runtime.mapassign
//goland:noinspection GoUnusedParameter
func mapassign(t *rt.GoMapType, h *rt.GoMap, key unsafe.Pointer) unsafe.Pointer

//go:noescape
//go:linkname mapassign_fast32 runtime.mapassign_fast32
//goland:noinspection GoUnusedParameter
func mapassign_fast32(t *rt.GoMapType, h *rt.GoMap, key uint32) unsafe.Pointer

//go:noescape
//go:linkname mapassign_fast64 runtime.mapassign_fast64
//goland:noinspection GoUnusedParameter
func mapassign_fast64(t *rt.GoMapType, h *rt.GoMap, key uint64) unsafe.Pointer

//go:noescape
//go:linkname mapassign_faststr runtime.mapassign_faststr
//goland:noinspection GoUnusedParameter
func mapassign_faststr(t *rt.GoMapType, h *rt.GoMap, s string) unsafe.Pointer

//go:noescape
//go:linkname mapassign_fast64ptr runtime.mapassign_fast64ptr
//goland:noinspection GoUnusedParameter
func mapassign_fast64ptr(t *rt.GoMapType, h *rt.GoMap, key unsafe.Pointer) unsafe.Pointer
