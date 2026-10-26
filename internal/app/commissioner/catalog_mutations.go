package commissioner

import (
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
)

func NativeCommands() []catalog.Command {
	return catalog.NativeCommands(catalog.Shape(reflect.TypeFor[RawNativeSeedRequest](), "RawNativeSeedRequest.Validate/ValidatePreview"))
}
