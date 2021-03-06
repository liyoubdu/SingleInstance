package singleInstance

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var lockedByThis = false

//HelloTest is fun
func HelloTest() {
	fmt.Printf("singleInstance say hello to you!\n")
}

//CurrentProcessIsSingle is fun
func CurrentProcessIsSingle(singleKey, lockFileName string) (singling bool, err error) {
	if "" == lockFileName {
		lockFileName = "lock.txt"
	}
	if len(lockFileName) < 5 || len(lockFileName) > 20 {
		return false, fmt.Errorf("invalid length of lockFileName")
	}
	if len(singleKey) < 5 || len(singleKey) > 30 {
		return false, fmt.Errorf("invalid length of singleKey")
	}
	locked, newLocker := locked(singleKey)
	if !locked {
		return false, nil
	}
	if !newLocker {
		return true, nil
	}

	//we get new locker, update time to file
	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	file, err := os.OpenFile(exeDir+"\\"+lockFileName, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("can not open pid.txt file: %s", err)
		}
		//no file exist, so we create new file
		file, err = os.OpenFile(exeDir+"\\"+lockFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return false, fmt.Errorf("can not open pid.txt file")
		}
	}

	data := fmt.Sprintf("[%s] [pid=%d]\n", time.Now().String(), os.Getpid())
	n, err := file.WriteString(data)
	if err != nil || n != len(data) {
		return true, fmt.Errorf("can not write string to pid.txt file")
	}

	//hold locker file
	go func() {
		for {
			data := make([]byte, 8)
			file.ReadAt(data, 0)
			time.Sleep(time.Hour)
		}
	}()

	return true, nil
}

func locked(key string) (locked, newLocker bool) {
	if false == lockedByThis {
		//test for new locker
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		procCreateMutex := kernel32.NewProc("CreateMutexW")
		closeHandle := kernel32.NewProc("CloseHandle")

		//call CreateMutex
		handle, _, err := procCreateMutex.Call(
			0,
			1,
			uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(key))),
		)

		//fmt.Printf("CreateMutexW, handle=%d errInt=%d errStr=%v [singleInstance]\n", handle, int(err.(syscall.Errno)), err)

		//check return val and last err
		if 0 == int(err.(syscall.Errno)) {
			lockedByThis = true
			locked = true
			newLocker = true
		} else { //fail to get locker, we have to release reference count of the kernel object.
			if handle != 0 {
				closeHandle.Call(handle)
			}
			lockedByThis = false
			locked = false
			newLocker = false
		}

		return
	}

	//we have keep this locker
	locked = true
	newLocker = false
	return
}
