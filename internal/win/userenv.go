package win

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// EnvValue is a raw registry value: Data is never expanded, Expand says it
// is REG_EXPAND_SZ, Exists false means there is no value at all.
type EnvValue struct {
	Exists bool   `json:"exists"`
	Data   string `json:"data,omitempty"`
	Expand bool   `json:"expand,omitempty"`
}

// UserEnv is the user's environment in HKEY_CURRENT_USER\<Key> ("" means
// "Environment"). Announce makes Broadcast tell running programs.
type UserEnv struct {
	Key      string
	Announce bool
}

func (u UserEnv) key() string {
	if u.Key == "" {
		return "Environment"
	}
	return u.Key
}

func (u UserEnv) Get(name string) (EnvValue, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, u.key(), registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return EnvValue{}, nil
	}
	if err != nil {
		return EnvValue{}, err
	}
	defer k.Close()
	s, typ, err := k.GetStringValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return EnvValue{}, nil
	}
	if err != nil {
		return EnvValue{}, err
	}
	return EnvValue{Exists: true, Data: s, Expand: typ == registry.EXPAND_SZ}, nil
}

func (u UserEnv) Set(name string, v EnvValue) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, u.key(), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if v.Expand {
		return k.SetExpandStringValue(name, v.Data)
	}
	return k.SetStringValue(name, v.Data)
}

func (u UserEnv) Delete(name string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, u.key(), registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

var procSendMessageTimeout = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")

// Broadcast tells running programs (Explorer first) that the environment
// changed, waiting at most 5 s for any that hangs.
func (u UserEnv) Broadcast() {
	if !u.Announce {
		return
	}
	const hwndBroadcast, wmSettingChange, smtoAbortIfHung = 0xffff, 0x001a, 0x0002
	env, _ := windows.UTF16PtrFromString("Environment")
	var result uintptr
	procSendMessageTimeout.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}
