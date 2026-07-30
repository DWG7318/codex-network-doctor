//go:build windows

package proxyrepair

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

type registryValueStore struct {
	key registry.Key
}

func openRegistryValueStore(root registry.Key, path string) (*registryValueStore, error) {
	key, _, err := registry.CreateKey(root, path, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return nil, err
	}
	return &registryValueStore{key: key}, nil
}

func (store *registryValueStore) Close() error {
	return store.key.Close()
}

func (store *registryValueStore) Read(name string) (SettingValue, error) {
	_, valueType, err := store.key.GetValue(name, nil)
	if errors.Is(err, registry.ErrNotExist) {
		return SettingValue{Present: false}, nil
	}
	if err != nil {
		return SettingValue{}, err
	}
	switch valueType {
	case registry.DWORD:
		value, _, err := store.key.GetIntegerValue(name)
		if err != nil {
			return SettingValue{}, err
		}
		return SettingValue{Present: true, Kind: "dword", DWord: uint32(value)}, nil
	case registry.SZ, registry.EXPAND_SZ:
		value, actualType, err := store.key.GetStringValue(name)
		if err != nil {
			return SettingValue{}, err
		}
		kind := "string"
		if actualType == registry.EXPAND_SZ {
			kind = "expand-string"
		}
		return SettingValue{Present: true, Kind: kind, String: value}, nil
	default:
		return SettingValue{}, fmt.Errorf("unsupported registry type %d for %s", valueType, name)
	}
}

func (store *registryValueStore) Write(name string, value SettingValue) error {
	if !value.Present {
		return fmt.Errorf("cannot write absent value %s", name)
	}
	switch value.Kind {
	case "dword":
		return store.key.SetDWordValue(name, value.DWord)
	case "string":
		return store.key.SetStringValue(name, value.String)
	case "expand-string":
		return store.key.SetExpandStringValue(name, value.String)
	default:
		return fmt.Errorf("unsupported setting kind %q for %s", value.Kind, name)
	}
}

func (store *registryValueStore) Delete(name string) error {
	err := store.key.DeleteValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}
