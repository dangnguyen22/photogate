package main

import (
	"fmt"
	"syscall"
)

func init() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("can't get rlimit: ", err)
	}
	fmt.Println("current nofile =", rLimit.Cur)

	if rLimit.Cur < 10000 {
		if rLimit.Max < 10000 {
			rLimit.Cur = rLimit.Max
		} else {
			rLimit.Cur = 10000
		}

		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			fmt.Println("can't set rlimit ", err)
		} else {
			fmt.Println("updated nofile =", rLimit.Cur)
		}
	}
}
