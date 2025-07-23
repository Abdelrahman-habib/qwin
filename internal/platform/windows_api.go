//go:build windows

package platform

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	psapi                        = windows.NewLazySystemDLL("psapi.dll")
	shell32                      = windows.NewLazySystemDLL("shell32.dll")
	gdi32                        = windows.NewLazySystemDLL("gdi32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetModuleFileNameExW     = psapi.NewProc("GetModuleFileNameExW")
	procExtractIconExW           = shell32.NewProc("ExtractIconExW")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procGetIconInfo              = user32.NewProc("GetIconInfo")
	procGetDIBits                = gdi32.NewProc("GetDIBits")
	procCreateCompatibleDC       = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC                 = gdi32.NewProc("DeleteDC")
	procDeleteObject             = gdi32.NewProc("DeleteObject")
)

type ICONINFO struct {
	fIcon    uint32
	xHotspot uint32
	yHotspot uint32
	hbmMask  syscall.Handle
	hbmColor syscall.Handle
}

type BITMAPINFOHEADER struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

// WindowsAPI implements WindowAPI for Windows platform
type WindowsAPI struct{}

// NewWindowsAPI creates a new Windows API instance
func NewWindowsAPI() *WindowsAPI {
	return &WindowsAPI{}
}

// NewWindowAPI creates a new WindowAPI instance for Windows
func NewWindowAPI() WindowAPI {
	return NewWindowsAPI()
}

// GetCurrentAppName gets the name of the currently active application
func (w *WindowsAPI) GetCurrentAppName() string {
	appInfo := w.GetCurrentAppInfo()
	if appInfo == nil {
		return ""
	}
	return appInfo.Name
}

// GetCurrentAppInfo gets detailed information about the currently active application
func (w *WindowsAPI) GetCurrentAppInfo() *AppInfo {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return nil
	}

	var processID uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	if processID == 0 {
		return nil
	}

	// Open process with PROCESS_QUERY_INFORMATION | PROCESS_VM_READ
	hProcess, _, _ := procOpenProcess.Call(0x0400|0x0010, 0, uintptr(processID))
	if hProcess == 0 {
		return nil
	}
	defer procCloseHandle.Call(hProcess)

	// Get the executable path
	var buffer [windows.MAX_PATH]uint16
	ret, _, _ := procGetModuleFileNameExW.Call(hProcess, 0, uintptr(unsafe.Pointer(&buffer[0])), windows.MAX_PATH)
	if ret == 0 {
		return nil
	}

	exePath := windows.UTF16ToString(buffer[:])
	if exePath == "" {
		return nil
	}

	// Extract just the filename without extension
	filename := filepath.Base(exePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Get icon path (for now, we'll use the exe path as icon source)
	iconPath := w.extractIconToTemp(exePath)

	return &AppInfo{
		Name:     name,
		IconPath: iconPath,
		ExePath:  exePath,
	}
}

// extractIconToTemp extracts the icon from an executable and returns it as base64 data URL
func (w *WindowsAPI) extractIconToTemp(exePath string) string {
	// Convert path to UTF16 for Windows API
	pathPtr, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return ""
	}

	// Extract large icon (32x32)
	var hIcon uintptr
	ret, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,                               // icon index
		uintptr(unsafe.Pointer(&hIcon)), // large icon
		0,                               // small icon (we don't need it)
		1,                               // number of icons to extract
	)

	if ret == 0 || hIcon == 0 {
		return ""
	}
	defer procDestroyIcon.Call(hIcon)

	// Get icon info
	var iconInfo ICONINFO
	ret, _, _ = procGetIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&iconInfo)))
	if ret == 0 {
		return ""
	}
	defer procDeleteObject.Call(uintptr(iconInfo.hbmColor))
	defer procDeleteObject.Call(uintptr(iconInfo.hbmMask))

	// Convert icon to base64 data URL
	dataURL := w.iconToDataURL(iconInfo.hbmColor)
	return dataURL
}

// iconToDataURL converts a Windows bitmap handle to a base64 data URL
func (w *WindowsAPI) iconToDataURL(hBitmap syscall.Handle) string {
	// Create compatible DC
	hdc, _, _ := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		return ""
	}
	defer procDeleteDC.Call(hdc)

	// Get bitmap info
	var bmi BITMAPINFOHEADER
	bmi.biSize = uint32(unsafe.Sizeof(bmi))

	// Get bitmap dimensions
	ret, _, _ := procGetDIBits.Call(
		hdc,
		uintptr(hBitmap),
		0, 0,
		0, // lpvBits = NULL to get info only
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
	)
	if ret == 0 {
		return ""
	}

	// Allocate buffer for bitmap data
	width := int(bmi.biWidth)
	height := int(bmi.biHeight)
	if height < 0 {
		height = -height
	}

	// For 32-bit RGBA
	bmi.biBitCount = 32
	bmi.biCompression = 0 // BI_RGB
	bmi.biSizeImage = uint32(width * height * 4)

	buffer := make([]byte, bmi.biSizeImage)

	// Get actual bitmap data
	ret, _, _ = procGetDIBits.Call(
		hdc,
		uintptr(hBitmap),
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
	)
	if ret == 0 {
		return ""
	}

	// Convert BGRA to RGBA and create PNG
	img := w.createImageFromBGRA(buffer, width, height)
	if img == nil {
		return ""
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}

	// Convert to base64 data URL
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/png;base64,%s", encoded)
}

// createImageFromBGRA creates an image.Image from BGRA byte data
func (w *WindowsAPI) createImageFromBGRA(data []byte, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Windows bitmaps are stored bottom-up, so flip Y coordinate
			srcY := height - 1 - y
			srcOffset := (srcY*width + x) * 4
			dstOffset := (y*width + x) * 4

			if srcOffset+3 < len(data) && dstOffset+3 < len(img.Pix) {
				// Convert BGRA to RGBA
				img.Pix[dstOffset+0] = data[srcOffset+2] // R
				img.Pix[dstOffset+1] = data[srcOffset+1] // G
				img.Pix[dstOffset+2] = data[srcOffset+0] // B
				img.Pix[dstOffset+3] = data[srcOffset+3] // A
			}
		}
	}

	return img
}
