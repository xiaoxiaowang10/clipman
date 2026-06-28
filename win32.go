package clipman

import (
	"syscall"
	"unsafe"
)

const (
	WM_NULL    = 0x0000
	WM_DESTROY = 0x0002
	WM_COMMAND = 0x0111
	WM_HOTKEY  = 0x0312
	WM_CREATE  = 0x0001
	WM_APP     = 0x8000

	WM_TRAYICON          = WM_APP + 1
	NIM_ADD              = 0
	NIM_DELETE           = 2
	NIM_SETVERSION       = 4
	NIF_MESSAGE          = 1
	NIF_ICON             = 2
	NIF_TIP              = 4
	NOTIFYICON_VERSION_4 = 4

	ID_SHOW         = 1001
	ID_EXIT         = 1002
	HOTKEY_ID       = 1
	MOD_CONTROL     = 0x0002
	MOD_WIN         = 0x0008
	VK_V            = 0x56
	TPM_RIGHTBUTTON = 0x0002
	GMEM_MOVABLE    = 0x0002
	IDI_APPLICATION = 32512
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	procOpenClipboard       = user32.NewProc("OpenClipboard")
	procCloseClipboard      = user32.NewProc("CloseClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procSetClipboardData    = user32.NewProc("SetClipboardData")
	procEmptyClipboard      = user32.NewProc("EmptyClipboard")
	procGlobalLock          = kernel32.NewProc("GlobalLock")
	procGlobalUnlock        = kernel32.NewProc("GlobalUnlock")
	procGlobalAlloc         = kernel32.NewProc("GlobalAlloc")
	procMultiByteToWideChar = kernel32.NewProc("MultiByteToWideChar")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procRegisterHotKey      = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey    = user32.NewProc("UnregisterHotKey")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
)

type NOTIFYICONDATA struct {
	cbSize           uint32
	hWnd             syscall.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            syscall.Handle
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         syscall.GUID
	hBalloonIcon     syscall.Handle
}

type POINT struct {
	X, Y int32
}

type WNDCLASSEX struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

var (
	hInst    syscall.Handle
	hwndMain syscall.Handle
)

func getClipText() string {
	r, _, _ := procOpenClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer procCloseClipboard.Call()

	// try CF_UNICODETEXT first (Windows auto-synthesizes from CF_TEXT)
	h, _, _ := procGetClipboardData.Call(13)
	if h == 0 {
		// fallback: read CF_TEXT (ANSI) and convert via MultiByteToWideChar
		h, _, _ = procGetClipboardData.Call(1)
		if h == 0 {
			return ""
		}
		p, _, _ := procGlobalLock.Call(h)
		if p == 0 {
			return ""
		}
		defer procGlobalUnlock.Call(h)
		var buf []byte
		for i := 0; ; i++ {
			b := *(*byte)(unsafe.Pointer(p + uintptr(i)))
			buf = append(buf, b)
			if b == 0 {
				break
			}
		}
		// convert ANSI multi-byte → UTF-16 → Go string
		n, _, _ := procMultiByteToWideChar.Call(0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)-1), 0, 0)
		if n == 0 {
			return string(buf[:len(buf)-1])
		}
		ws := make([]uint16, n)
		procMultiByteToWideChar.Call(0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)-1), uintptr(unsafe.Pointer(&ws[0])), n)
		return syscall.UTF16ToString(ws)
	}

	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)

	var chars []uint16
	for i := 0; ; i++ {
		ch := *(*uint16)(unsafe.Pointer(p + uintptr(i*2)))
		chars = append(chars, ch)
		if ch == 0 {
			break
		}
	}
	return syscall.UTF16ToString(chars)
}

func setClipText(text string) {
	ownCopy.Store(true)
	utf16, _ := syscall.UTF16FromString(text)
	procOpenClipboard.Call(0)
	procEmptyClipboard.Call()
	h, _, _ := procGlobalAlloc.Call(GMEM_MOVABLE, uintptr(len(utf16)*2))
	if h != 0 {
		p, _, _ := procGlobalLock.Call(h)
		if p != 0 {
			copy((*[1 << 20]uint16)(unsafe.Pointer(p))[:len(utf16)], utf16)
		}
		procGlobalUnlock.Call(h)
	}
	procSetClipboardData.Call(13, h)
	procCloseClipboard.Call()
}

func addTrayIcon(hwnd syscall.Handle) {
	hIcon, _, _ := procLoadIconW.Call(0, IDI_APPLICATION)
	title, _ := syscall.UTF16FromString("Clipman")
	nid := NOTIFYICONDATA{
		cbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		hWnd:             hwnd,
		uID:              1,
		uFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		uCallbackMessage: WM_TRAYICON,
		hIcon:            syscall.Handle(hIcon),
	}
	copy(nid.szTip[:], title)
	procShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	nid.uFlags = 0
	nid.uVersion = NOTIFYICON_VERSION_4
	procShellNotifyIcon.Call(NIM_SETVERSION, uintptr(unsafe.Pointer(&nid)))
}

func removeTrayIcon(hwnd syscall.Handle) {
	nid := NOTIFYICONDATA{
		cbSize: uint32(unsafe.Sizeof(NOTIFYICONDATA{})),
		hWnd:   hwnd,
		uID:    1,
	}
	procShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func showTrayMenu(hwnd syscall.Handle) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	show, _ := syscall.UTF16FromString("显示搜索")
	exit, _ := syscall.UTF16FromString("退出")
	procAppendMenuW.Call(hMenu, 0, ID_SHOW, uintptr(unsafe.Pointer(&show[0])))
	procAppendMenuW.Call(hMenu, 0x0800, 0, 0)
	procAppendMenuW.Call(hMenu, 0, ID_EXIT, uintptr(unsafe.Pointer(&exit[0])))
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(uintptr(hwnd))
	procTrackPopupMenu.Call(hMenu, TPM_RIGHTBUTTON, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hwnd), 0)
	procPostMessageW.Call(uintptr(hwnd), WM_NULL, 0, 0)
	procDestroyMenu.Call(hMenu)
}

func mainWndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		addTrayIcon(hwnd)
		procRegisterHotKey.Call(uintptr(hwnd), HOTKEY_ID, MOD_CONTROL|MOD_WIN, VK_V)
		return 0
	case WM_TRAYICON:
		if (lparam & 0xFFFF) == 0x0205 {
			showTrayMenu(hwnd)
		}
		if (lparam & 0xFFFF) == 0x0203 {
			openBrowser()
		}
		return 0
	case WM_COMMAND:
		id := wparam & 0xFFFF
		if id == ID_SHOW {
			openBrowser()
		}
		if id == ID_EXIT {
			procDestroyWindow.Call(uintptr(hwnd))
		}
		return 0
	case WM_HOTKEY:
		openBrowser()
		return 0
	case WM_DESTROY:
		removeTrayIcon(hwnd)
		procUnregisterHotKey.Call(uintptr(hwnd), HOTKEY_ID)
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return ret
}

func createMainWindow(hInst syscall.Handle) syscall.Handle {
	const cls = "ClipmanMainClass"
	c, _ := syscall.UTF16FromString(cls)
	wc := WNDCLASSEX{
		cbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		lpfnWndProc:   syscall.NewCallback(mainWndProc),
		hInstance:     hInst,
		lpszClassName: &c[0],
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	t, _ := syscall.UTF16FromString("Clipman")
	h, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(&c[0])), uintptr(unsafe.Pointer(&t[0])), 0, 0, 0, 0, 0, 0, 0, uintptr(hInst), 0)
	return syscall.Handle(h)
}
