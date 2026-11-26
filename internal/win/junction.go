package win

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// CreateJunction makes a new folder at link that leads to target (a
// directory junction — no admin rights needed). It never replaces anything:
// an existing link path is an error.
func CreateJunction(link, target string) error {
	target, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		return err
	}
	if err := setMountPoint(link, target); err != nil {
		os.Remove(link)
		return err
	}
	return nil
}

func setMountPoint(link, target string) error {
	h, err := windows.CreateFile(windows.StringToUTF16Ptr(link), windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	sub := utf16.Encode([]rune(`\??\` + target))
	printName := utf16.Encode([]rune(target))
	pathBytes := (len(sub) + 1 + len(printName) + 1) * 2
	dataLen := 8 + pathBytes
	buf := make([]byte, 8+dataLen)
	le := binary.LittleEndian
	le.PutUint32(buf[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	le.PutUint16(buf[4:], uint16(dataLen))
	le.PutUint16(buf[8:], 0)                         // substitute name offset
	le.PutUint16(buf[10:], uint16(len(sub)*2))       // substitute name length
	le.PutUint16(buf[12:], uint16((len(sub)+1)*2))   // print name offset
	le.PutUint16(buf[14:], uint16(len(printName)*2)) // print name length
	off := 16
	for _, c := range sub {
		le.PutUint16(buf[off:], c)
		off += 2
	}
	off += 2
	for _, c := range printName {
		le.PutUint16(buf[off:], c)
		off += 2
	}
	var n uint32
	return windows.DeviceIoControl(h, windows.FSCTL_SET_REPARSE_POINT, &buf[0], uint32(len(buf)), nil, 0, &n, nil)
}

// JunctionTarget returns where the junction at link leads, even when the
// target is gone. Anything that is not a junction is an error.
func JunctionTarget(link string) (string, error) {
	h, err := windows.CreateFile(windows.StringToUTF16Ptr(link), 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	buf := make([]byte, windows.MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var n uint32
	if err := windows.DeviceIoControl(h, windows.FSCTL_GET_REPARSE_POINT, nil, 0, &buf[0], uint32(len(buf)), &n, nil); err != nil {
		return "", err
	}
	le := binary.LittleEndian
	if le.Uint32(buf[0:]) != windows.IO_REPARSE_TAG_MOUNT_POINT {
		return "", errors.New(link + " is not a junction")
	}
	off, size := int(le.Uint16(buf[8:])), int(le.Uint16(buf[10:]))
	name := make([]uint16, size/2)
	for i := range name {
		name[i] = le.Uint16(buf[16+off+2*i:])
	}
	return strings.TrimPrefix(string(utf16.Decode(name)), `\??\`), nil
}
