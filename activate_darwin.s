// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

#include "textflag.h"

TEXT    libc_launch_activate_socket_trampoline<>(SB),NOSPLIT,$0-0
	        JMP	libc_launch_activate_socket(SB)
            RET

TEXT    libc_free_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_free(SB)
            RET

GLOBL	·libc_launch_activate_socket_trampoline_addr(SB), RODATA, $8
GLOBL	·libc_free_trampoline_addr(SB), RODATA, $8

DATA	·libc_launch_activate_socket_trampoline_addr(SB)/8, $libc_launch_activate_socket_trampoline<>(SB)
DATA	·libc_free_trampoline_addr(SB)/8, $libc_free_trampoline<>(SB)
