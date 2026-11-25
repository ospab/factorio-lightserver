package main

import (
	"encoding/binary"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx       = kernel32.NewProc("GetDiskFreeSpaceExW")
	getConsoleProcessList    = kernel32.NewProc("GetConsoleProcessList")
	queryImageName           = kernel32.NewProc("QueryFullProcessImageNameW")
	getProcessMemoryInfo     = kernel32.NewProc("K32GetProcessMemoryInfo")
	getUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
	getExtendedUDPTable      = syscall.NewLazyDLL("iphlpapi.dll").NewProc("GetExtendedUdpTable")
)

const processQueryLimited, processSynchronize = 0x1000, 0x100000

func freeDisk(dir string) (uint64, error) {
	p, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var free uint64
	r, _, err := getDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&free)), 0, 0)
	if r == 0 {
		return 0, err
	}
	return free, nil
}

// readRegistry returns a string value from HKEY_LOCAL_MACHINE, or "".
func readRegistry(key, value string) string {
	k, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return ""
	}
	v, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return ""
	}
	var h syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, k, 0, syscall.KEY_READ, &h) != nil {
		return ""
	}
	defer syscall.RegCloseKey(h)
	var buf [1024]uint16
	var kind uint32
	size := uint32(len(buf) * 2)
	if syscall.RegQueryValueEx(h, v, nil, &kind, (*byte)(unsafe.Pointer(&buf[0])), &size) != nil || kind != syscall.REG_SZ {
		return ""
	}
	return syscall.UTF16ToString(buf[:size/2])
}

// udpPortUsed reads the system table instead of opening a socket: a socket
// would make the firewall ask about flserver.exe as well.
func udpPortUsed(port int) bool {
	const afInet, udpTableBasic = 2, 0
	buf := make([]byte, 64<<10)
	for range 4 {
		size := uint32(len(buf))
		r, _, _ := getExtendedUDPTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, afInet, udpTableBasic, 0)
		if syscall.Errno(r) == syscall.ERROR_INSUFFICIENT_BUFFER {
			buf = make([]byte, size+4096)
			continue
		}
		if r != 0 {
			return false
		}
		n := int(binary.LittleEndian.Uint32(buf))
		for i := range n { // a row is the address and the port in network order, 4 bytes each
			if int(binary.BigEndian.Uint16(buf[4+i*8+4:])) == port {
				return true
			}
		}
		return false
	}
	return false
}

// ownConsole reports whether the console window closes together with us, as
// it does after a double click.
func ownConsole() bool {
	var ids [2]uint32
	n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&ids[0])), 2)
	return n == 1
}

// processAlive checks that the PID is alive and is factorio.exe: Windows may
// have given the number to another program.
func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimited|processSynchronize, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	if ev, _ := syscall.WaitForSingleObject(h, 0); ev != syscall.WAIT_TIMEOUT {
		return false
	}
	var buf [1024]uint16
	n := uint32(len(buf))
	r, _, _ := queryImageName.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	return r != 0 && strings.EqualFold(filepath.Base(syscall.UTF16ToString(buf[:n])), "factorio.exe")
}

// workingSet returns the physical memory a process holds, in bytes.
func workingSet(pid int) uint64 {
	h, err := syscall.OpenProcess(processQueryLimited, false, uint32(pid))
	if err != nil {
		return 0
	}
	defer syscall.CloseHandle(h)
	var m struct {
		cb, faults                                         uint32
		peakWS, ws, peakPP, pp, peakNP, np, page, peakPage uintptr
	}
	m.cb = uint32(unsafe.Sizeof(m))
	getProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&m)), uintptr(m.cb))
	return uint64(m.ws)
}
