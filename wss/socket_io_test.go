package wss

import (
	"testing"
)

func TestSocketIO_BuildEventPacket(t *testing.T) {
	sio := NewSocketIO()

	// 无 ackId
	pkt := sio.BuildEventPacket("chat", []byte(`"hello"`))
	want := `2/chat,"hello"`
	if string(pkt) != want {
		t.Errorf("BuildEventPacket: got %q, want %q", pkt, want)
	}

	// 有 ackId
	pkt = sio.BuildEventPacket("chat", []byte(`"hello"`), 42)
	want = `2/chat,42,"hello"`
	if string(pkt) != want {
		t.Errorf("BuildEventPacket with ack: got %q, want %q", pkt, want)
	}
}

func TestSocketIO_BuildAckPacket(t *testing.T) {
	sio := NewSocketIO()

	pkt := sio.BuildAckPacket(123, []byte(`"ok"`))
	want := `3/123,"ok"`
	if string(pkt) != want {
		t.Errorf("BuildAckPacket: got %q, want %q", pkt, want)
	}
}

func TestSocketIO_BuildConnectPacket(t *testing.T) {
	sio := NewSocketIO()

	// 默认命名空间
	pkt := sio.BuildConnectPacket()
	want := "0//"
	if string(pkt) != want {
		t.Errorf("BuildConnectPacket default: got %q, want %q", pkt, want)
	}

	// 自定义命名空间
	pkt = sio.BuildConnectPacket("/chat")
	want = "0//chat"
	if string(pkt) != want {
		t.Errorf("BuildConnectPacket /chat: got %q, want %q", pkt, want)
	}
}

func TestSocketIO_BuildDisconnectPacket(t *testing.T) {
	sio := NewSocketIO()

	pkt := sio.BuildDisconnectPacket()
	want := "1//"
	if string(pkt) != want {
		t.Errorf("BuildDisconnectPacket: got %q, want %q", pkt, want)
	}
}

func TestSocketIO_BuildBinaryEventPacket(t *testing.T) {
	sio := NewSocketIO()

	// 无 ackId
	pkt := sio.BuildBinaryEventPacket(1, "upload", []byte(`"data"`))
	want := `5-1/upload,"data"`
	if string(pkt) != want {
		t.Errorf("BuildBinaryEventPacket: got %q, want %q", pkt, want)
	}

	// 有 ackId
	pkt = sio.BuildBinaryEventPacket(2, "upload", []byte(`"data"`), 99)
	want = `5-2/upload,99,"data"`
	if string(pkt) != want {
		t.Errorf("BuildBinaryEventPacket with ack: got %q, want %q", pkt, want)
	}
}

func TestSocketIO_CustomNamespace(t *testing.T) {
	sio := NewSocketIO("/game")

	if sio.Namespace != "/game" {
		t.Errorf("Namespace: got %q, want /game", sio.Namespace)
	}

	pkt := sio.BuildConnectPacket()
	want := "0//game"
	if string(pkt) != want {
		t.Errorf("BuildConnectPacket /game: got %q, want %q", pkt, want)
	}
}
