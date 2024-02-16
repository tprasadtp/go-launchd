// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

#include "textflag.h"

TEXT    libc_trampoline_launch_activate_socket<>(SB),NOSPLIT,$0-0
	        JMP	libc_launch_activate_socket(SB)
            RET

TEXT    libc_trampoline_free<>(SB),NOSPLIT,$0-0
            JMP	libc_free(SB)
            RET

GLOBL	·libc_trampoline_launch_activate_socket_addr(SB), RODATA, $8
GLOBL	·libc_trampoline_free_addr(SB), RODATA, $8

DATA	·libc_trampoline_launch_activate_socket_addr(SB)/8, $libc_trampoline_launch_activate_socket<>(SB)
DATA	·libc_trampoline_free_addr(SB)/8, $libc_trampoline_free<>(SB)
