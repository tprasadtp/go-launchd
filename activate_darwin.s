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

// Following APIs are deprecated, but there is no equivalent.
// Use them anyways. Chrome also does the same thing.
//
// https://chromium.googlesource.com/chromium/src/+/987a6cf1eebd29cfc605e9cee39a30ab5fe613b2/base/mac/launchd.cc#11

GLOBL	·libc_launch_msg_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_msg_trampoline_addr(SB)/8, $libc_launch_msg_trampoline<>(SB)
TEXT    launch_msg_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_msg(SB)
            RET


GLOBL	·libc_launch_data_get_errno_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_get_errno_trampoline_addr(SB)/8, $libc_launch_data_get_errno_trampoline<>(SB)
TEXT    launch_data_get_errno_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_get_errno(SB)
            RET

GLOBL	·libc_launch_data_alloc_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_alloc_trampoline_addr(SB)/8, $libc_launch_data_alloc_trampoline<>(SB)
TEXT    launch_data_alloc_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_alloc(SB)
            RET

GLOBL	·libc_launch_data_free_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_free_trampoline_addr(SB)/8, $libc_launch_data_free_trampoline<>(SB)
TEXT    launch_data_free_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_free(SB)
            RET

GLOBL	·libc_launch_data_new_string_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_new_string_trampoline_addr(SB)/8, $libc_launch_data_new_string_trampoline<>(SB)
TEXT    launch_data_new_string_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_new_string(SB)
            RET

GLOBL	·libc_launch_data_get_string_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_get_string_trampoline_addr(SB)/8, $libc_launch_data_get_string_trampoline<>(SB)
TEXT    launch_data_get_string_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_get_string(SB)
            RET

GLOBL	·libc_launch_data_get_type_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_get_type_trampoline_addr(SB)/8, $libc_launch_data_get_type_trampoline<>(SB)
TEXT    launch_data_get_type_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_get_type(SB)
            RET

GLOBL	·libc_launch_data_get_integer_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_get_integer_trampoline_addr(SB)/8, $libc_launch_data_get_integer_trampoline<>(SB)
TEXT    launch_data_get_integer_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_get_integer(SB)
            RET

GLOBL	·libc_launch_data_dict_lookup_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_dict_lookup_trampoline_addr(SB)/8, $libc_launch_data_dict_lookup_trampoline<>(SB)
TEXT    launch_data_dict_lookup_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_dict_lookup(SB)
            RET

GLOBL	·libc_launch_data_array_get_count_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_array_get_count_trampoline_addr(SB)/8, $libc_launch_data_array_get_count_trampoline<>(SB)
TEXT    launch_data_array_get_count_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_array_get_count(SB)
            RET

GLOBL	·libc_launch_data_array_get_index_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_array_get_index_trampoline_addr(SB)/8, $libc_launch_data_array_get_index_trampoline<>(SB)
TEXT    launch_data_array_get_index_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_array_get_index(SB)
            RET

// Not used but is here for reference.
GLOBL	·libc_launch_data_get_fd_trampoline_addr(SB), RODATA, $8
DATA	·libc_launch_data_get_fd_trampoline_addr(SB)/8, $libc_launch_data_get_fd_trampoline<>(SB)
TEXT    launch_data_get_fd_trampoline<>(SB),NOSPLIT,$0-0
            JMP	libc_launch_data_get_fd(SB)
            RET
