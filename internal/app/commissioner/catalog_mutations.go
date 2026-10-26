package commissioner

import (
	"reflect"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func NativeCommands() []catalog.Command {
	return catalog.NativeCommands(catalog.Shape(reflect.TypeFor[RawNativeSeedRequest](), "RawNativeSeedRequest.Validate/ValidatePreview"))
}
