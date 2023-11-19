// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

#include "textflag.h"

// This is simply an assembly trampoline to call library routines from Go
// without using cgo.
GLOBL	·libxpc_launch_activate_socket_trampoline_addr(SB), RODATA, $8
DATA	·libxpc_launch_activate_socket_trampoline_addr(SB)/8, $libxpc_launch_activate_socket_trampoline<>(SB)
TEXT    libxpc_launch_activate_socket_trampoline<>(SB),NOSPLIT,$0-0
	        JMP	libxpc_launch_activate_socket(SB)
            RET

GLOBL	·libc_free_trampoline_addr(SB), RODATA, $8
DATA	·libc_free_trampoline_addr(SB)/8, $libc_free_trampoline<>(SB)
TEXT    libc_free_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_free(SB)
            RET
