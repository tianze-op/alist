// Code generated by 'yaegi extract github.com/alist-org/alist/v3/internal/plugin/yaegi/adapter/storage'. DO NOT EDIT.

package lib

import (
	"github.com/alist-org/alist/v3/internal/plugin/yaegi/adapter/storage"
	"reflect"
)

func init() {
	Symbols["github.com/alist-org/alist/v3/internal/plugin/yaegi/adapter/storage/yaegi_storage"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"DropPluginStorage":      reflect.ValueOf(yaegi_storage.DropPluginStorage),
		"LoadPluginStorage":      reflect.ValueOf(yaegi_storage.LoadPluginStorage),
		"RegisterPluginDriver":   reflect.ValueOf(yaegi_storage.RegisterPluginDriver),
		"UnRegisterPluginDriver": reflect.ValueOf(yaegi_storage.UnRegisterPluginDriver),

		// type definitions
		"DriverPluginHelper_V1": reflect.ValueOf((*yaegi_storage.DriverPluginHelper_V1)(nil)),
		"ObjectHelper":          reflect.ValueOf((*yaegi_storage.ObjectHelper)(nil)),
	}
}
