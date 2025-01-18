package message

var Transform transform = transformDefault{}

type transform interface {
	Path(uint16) (string, error) //code to path
	Code(string) (uint16, error) //path to code
}

type transformDefault struct{}

func (t transformDefault) Path(uint16) (path string, err error) {
	err = ErrMsgHeadNotSetTransform
	return
}
func (t transformDefault) Code(string) (code uint16, err error) {
	err = ErrMsgHeadNotSetTransform
	return
}
