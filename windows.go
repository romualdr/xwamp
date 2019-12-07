package main

import (
	"github.com/lxn/win"
	"github.com/mitchellh/go-ps"
	"github.com/sqweek/dialog"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var (
	// Z
	diskLetter string
	// Z:
	vDisk string
	// Z:\
	vDiskPath string
	pwd, _ = os.Getwd()

	vDrivePath = `\vdrive`
	registryKey = `SOFTWARE\romualdr\xwamp\CurrentVersion`
	pathOperationMutex = sync.Mutex{}
)

func fatal(message string) {
	dialog.Message("%s", message).Title("Fatal error").Error()
}

func getOrCreateRegistry(keyName string) *registry.Key {
	key, err := registry.OpenKey(registry.CURRENT_USER, keyName, registry.ALL_ACCESS)
	if err == nil {
		return &key
	}
	if err != syscall.ERROR_FILE_NOT_FOUND {
		fatal("Unable to open registry")
		return nil
	}
	key, _, err = registry.CreateKey(registry.CURRENT_USER, keyName, registry.ALL_ACCESS)
	if err != nil {
		fatal("Unable to create registry")
	}
	return &key
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func getWindowsPointer(str string) *uint16 {
	ptr, err := syscall.UTF16PtrFromString(str)
	if err != nil {
		fatal("Unable to get string pointer for " + str)
	}
	return ptr
}

func getUnusedDrive() string {
	drives := strings.Split("ZYXWVUTSRQPONMLKJIHGFEDCBA", "")
	for _, s := range drives {
		diskLetter = s
		vDisk = diskLetter + `:`
		vDiskPath = vDisk + `\`
		if !exists(vDiskPath) {
			return diskLetter
		}
	}
	fatal("Unable to find an unused drive")
	return ""
}

func vDiskExists(vDisk string) bool {
	return exists(vDisk + `\`)
}

func getVDriveDirectory() string {
	return pwd + vDrivePath
}

func createDrive() {
	getUnusedDrive()
	dir := getVDriveDirectory()
	err := windows.DefineDosDevice(0, getWindowsPointer(vDisk), getWindowsPointer(dir))
	if err != nil {
		fatal("Unable to create dos device")
		return
	}
	setStringValue("vDisk", vDisk)
	println("Created " + vDiskPath + " pointing on " + dir)
}

func removeDrive(vDisk string) {
	vDiskPath := vDisk + `\`
	if !exists(vDiskPath) {
		fatal("VDisk " + vDisk + " does not exists")
		return
	}
	err := windows.DefineDosDevice(windows.DDD_REMOVE_DEFINITION, getWindowsPointer(vDisk), getWindowsPointer(getVDriveDirectory()))
	if err != nil {
		fatal("Unable to delete vdrive " + vDisk)
		return
	}
	println("Removed virtual drive " + vDiskPath)
}

func setCurrentDirectory(directory string) {
	err := windows.SetCurrentDirectory(getWindowsPointer(directory))
	if err != nil {
		fatal(err.Error())
	}
}

func getPIDs(exe string) []int {
	processList, err := ps.Processes()
	if err != nil {
		fatal("Unable to load process list")
	}
	var process ps.Process
	var count = 0
	for x := range processList {
		process = processList[x]
		if strings.EqualFold(process.Executable(), exe) {
			count = count + 1
		}
	}
	var index = 0
	var pids = make([]int, count)
	for x := range processList {
		process = processList[x]
		if strings.EqualFold(process.Executable(), exe) {
			pids[index] = process.Pid()
			index = index + 1
		}
	}

	return pids
}

func execute(path string, args string, cwd string, show bool) {
	var err error
	if show {
		err = windows.ShellExecute(0, getWindowsPointer("open"), getWindowsPointer(path), getWindowsPointer(args), getWindowsPointer(cwd), windows.SW_SHOW)
	} else {
		err = windows.ShellExecute(0, getWindowsPointer("open"), getWindowsPointer(path), getWindowsPointer(args), getWindowsPointer(cwd), windows.SW_HIDE)
	}
	if err != nil {
		fatal("Unable to open " + path)
	}
}

func _terminate(wg *sync.WaitGroup, pid int) error {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE | windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		fatal("Unable to open process " + strconv.Itoa(pid))
		return err
	}

	defer windows.CloseHandle(handle)
	defer wg.Done()

	win.EnumChildWindows(win.GetActiveWindow(), syscall.NewCallback(func(handle2 win.HWND, pid uint32) uintptr {
		var proc uint32
		win.GetWindowThreadProcessId(handle2, &proc)
		if proc == pid {
			win.PostMessage(handle2, win.WM_CLOSE, 0, 0)
		}
		return 1
	}), uintptr(pid))

	event, err := windows.WaitForSingleObject(handle, 5000)
	if err != nil {
		fatal("Unable to wait process " + strconv.Itoa(pid))
		return err
	}
	if event == syscall.WAIT_OBJECT_0 {
		return nil
	}

	err = windows.TerminateProcess(handle, 0)
	if err != nil {
		fatal("Unable to terminate process " + strconv.Itoa(pid))
		return err
	}
	event, err = windows.WaitForSingleObject(handle, 5000)
	if err != nil {
		fatal("Unable to wait process " + strconv.Itoa(pid))
		return err
	}
	return nil
}

func terminateAll(wg *sync.WaitGroup, exec string) {
	pids := getPIDs(exec)
	for x := range pids {
		wg.Add(1)
		go func(pid int) {
			err := _terminate(wg, pid)
			if err != nil {
				fatal("Unable to terminate " + exec + "[PID: " + strconv.Itoa(pid) + "]")
			}
		}(pids[x])
	}
}

func terminate(exec string) {
	var wg sync.WaitGroup
	terminateAll(&wg, exec)
	wg.Wait()
}

func setStringValue(name string, value string) {
	regedit := getOrCreateRegistry(registryKey)
	defer regedit.Close()
	err := regedit.SetStringValue(name, value)
	if err != nil {
		fatal("Unable to set " + name + " value")
	}
}

func setIntegerValue(name string, value int) {
	regedit := getOrCreateRegistry(registryKey)
	defer regedit.Close()
	err := regedit.SetQWordValue(name, uint64(value))
	if err != nil {
		fatal("Unable to set " + name + " value")
	}
}

func hasStringValue(name string) bool {
	regedit := getOrCreateRegistry(registryKey)
	defer regedit.Close()
	value, _, err := regedit.GetStringValue(name)
	return err == nil && len(value) > 0
}


func hasIntegerValue(name string) bool {
	regedit := getOrCreateRegistry(registryKey)
	defer regedit.Close()
	_, _, err := regedit.GetIntegerValue(name)
	return err == nil
}

func getStringValue(name string) string {
	regedit := getOrCreateRegistry(registryKey)
	defer regedit.Close()
	value, _, err := regedit.GetStringValue(name)
	if err != nil && err != syscall.ENOENT {
		fatal("Unable to get " + name + " value from registry")
	} else if len(value) > 0 {
		return value
	}
	return ""
}

func getIntegerValue(name string) int {
	regedit := getOrCreateRegistry(registryKey)
	defer regedit.Close()
	value, _, err := regedit.GetIntegerValue(name)
	if err != nil && err != syscall.ENOENT {
		fatal("Unable to get " + name + " value from registry")
	} else {
		return int(value)
	}
	return -1
}

func open(file string) {
	exec.Command("rundll32", "url.dll,FileProtocolHandler", file).Start()
}

func getPath() string {
	return os.Getenv("Path")
}

func addToPath(toAdd string) {
	pathOperationMutex.Lock()
	defer pathOperationMutex.Unlock()
	path := getPath()
	if strings.Contains(path, toAdd) {
		return
	}
	setPath(toAdd + ";" + path)
}

func removePath(toRemove string) {
	pathOperationMutex.Lock()
	defer pathOperationMutex.Unlock()
	path := getPath()
	index := strings.Index(path, toRemove)
	if index == -1 { return }
	path = strings.Replace(path, toRemove + ";", "", -1)
	setPath(path)
}

func setPath(path string) {
	err := os.Setenv("Path", path)
	if err != nil {
		fatal("Unable to set path environment variable")
	}
}

func openInEditor(path string) {
	execute("notepad.exe", path, "", true)
}
