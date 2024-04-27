// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

//go:build darwin && !ios

#include "textflag.h"

GLOBL	·libc_trampoline_launch_activate_socket_addr(SB), RODATA, $8
DATA	·libc_trampoline_launch_activate_socket_addr(SB)/8, $libc_trampoline_launch_activate_socket<>(SB)
TEXT    libc_trampoline_launch_activate_socket<>(SB),NOSPLIT,$0-0
	        JMP	libc_launch_activate_socket(SB)

GLOBL	·libc_trampoline_free_addr(SB), RODATA, $8
DATA	·libc_trampoline_free_addr(SB)/8, $libc_trampoline_free<>(SB)
TEXT    libc_trampoline_free<>(SB),NOSPLIT,$0-0
            JMP	libc_free(SB)
