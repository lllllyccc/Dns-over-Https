package server

import "syscall"

type syscallStatfs = syscall.Statfs_t

var syscallStatfsFn = syscall.Statfs
