package message

var Transform transform = transformDefault{}

type transform interface {
	Path(int32) (string, error) //code to path
	Code(string) (int32, error) //path to code
}

type transformDefault struct{}

func (t transformDefault) Path(int32) (path string, err error) {
	err = ErrMsgHeadNotSetTransform
	return
}
func (t transformDefault) Code(string) (code int32, err error) {
	err = ErrMsgHeadNotSetTransform
	return
}
