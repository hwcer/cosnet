package cosnet

import (
	"errors"
	"io"
	"net"
	"syscall"
	"testing"
)

// TestIsDisconnectError 对端断开类错误必须被识别,其余不得误伤。
//
// 🔴 Windows 陷阱(见 isDisconnectError 注释):syscall.ECONNRESET 是
// APPLICATION_ERROR 段(536870935),与真实网络错误的 WSAECONNRESET(10054)
// 数值不同,errors.Is 永远匹配不上——本用例在 Windows 上就是防"改成
// errors.Is(err, syscall.ECONNRESET) 就够了"这类回退的。
func TestIsDisconnectError(t *testing.T) {
	//用户线上实际形态:net.OpError 包装 WSAECONNRESET(10054)
	reset := &net.OpError{Op: "read", Net: "tcp", Err: syscall.Errno(10054)}
	aborted := &net.OpError{Op: "read", Net: "tcp", Err: syscall.Errno(10053)}
	cases := []error{
		io.EOF,
		io.ErrUnexpectedEOF,
		net.ErrClosed,
		reset,
		aborted,
	}
	for _, e := range cases {
		if !isDisconnectError(e) {
			t.Errorf("应识别为对端断开: %v", e)
		}
	}
	//非断开类错误不得被误过滤(仍会上报 EventTypeError)
	other := []error{
		errors.New("some real error"),
		&net.OpError{Op: "read", Net: "tcp", Err: errors.New("buffer overflow")},
	}
	for _, e := range other {
		if isDisconnectError(e) {
			t.Errorf("不应识别为对端断开: %v", e)
		}
	}
}
