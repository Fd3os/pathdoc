package win

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// MakePrivateDir creates dir so only the current user, SYSTEM and
// Administrators can use it. The folder is born with that DACL (no window in
// between) and inherits nothing: C:\ gives every authenticated user Modify,
// and a TEMP or Gradle home other accounts can write into lets them plant
// files that this user's builds and installers run.
func MakePrivateDir(dir string) error {
	user, err := currentUser()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.String() + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if err != nil {
		return err
	}
	sa := &windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	return windows.CreateDirectory(p, sa)
}

// IsPrivate reports whether path's DACL is protected from inheritance and
// allows only the current user, SYSTEM and Administrators.
func IsPrivate(path string) (bool, error) {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return false, err
	}
	ctrl, _, err := sd.Control()
	if err != nil || ctrl&windows.SE_DACL_PROTECTED == 0 {
		return false, err
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return false, err
	}
	user, err := currentUser()
	if err != nil {
		return false, err
	}
	system, _ := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	admins, _ := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	for i := 0; i < int(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			return false, err
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.Equals(user) && !sid.Equals(system) && !sid.Equals(admins) {
			return false, nil
		}
	}
	return true, nil
}

func currentUser() (*windows.SID, error) {
	tu, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return tu.User.Sid.Copy()
}
