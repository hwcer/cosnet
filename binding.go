package cosnet

import "github.com/hwcer/cosgo/binder"

type BindingType uint8

const (
	BindingTypeDefault BindingType = iota
	BindingTypeNumber
	BindingTypeString
	BindingTypeXml
	BindingTypeJson
	BindingTypeYaml
	BindingTypePostForm
	BindingTypeProtoBuff
)

var bindingMimeType = make(map[BindingType]string)
var bindingTypeDefault BindingType = BindingTypeJson

func init() {
	bindingMimeType[BindingTypeXml] = binder.MIMEXML
	bindingMimeType[BindingTypeJson] = binder.MIMEJSON
	bindingMimeType[BindingTypeYaml] = binder.MIMEYAML
	bindingMimeType[BindingTypePostForm] = binder.MIMEPOSTForm
	bindingMimeType[BindingTypeProtoBuff] = binder.MIMEPROTOBUF
}

// SetBindingType 设置默认绑定类型
func SetBindingType(t BindingType) {
	bindingTypeDefault = t
}

func GetBindingType(t BindingType) string {
	if t == BindingTypeDefault {
		t = bindingTypeDefault
	}
	return bindingMimeType[t]
}
