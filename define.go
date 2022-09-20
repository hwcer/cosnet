package cosnet

import "github.com/hwcer/cosgo/binding"

type NetType uint8

// NetType
const (
	NetTypeClient NetType = 1 //client Request
	NetTypeServer         = 2 //Server Listener
)

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
	bindingMimeType[BindingTypeXml] = binding.MIMEXML
	bindingMimeType[BindingTypeJson] = binding.MIMEJSON
	bindingMimeType[BindingTypeYaml] = binding.MIMEYAML
	bindingMimeType[BindingTypePostForm] = binding.MIMEPOSTForm
	bindingMimeType[BindingTypeProtoBuff] = binding.MIMEPROTOBUF
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
