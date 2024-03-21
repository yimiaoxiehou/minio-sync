package message

import "math/bits"

type RespMessage struct {
	Seq int32
	Ok  bool
}

func (x *RespMessage) EncodeToByte() byte {
	res := byte(uint8(x.Seq)) << 1
	if x.Ok {
		res = res | 1
	}
	return res
}

func DecodeFromByte(b byte) *RespMessage {
	// 后缀 0 数量大于 0
	ok := bits.TrailingZeros8(b) == 0
	seq := int32(uint8(b >> 1))
	return &RespMessage{
		Seq: seq,
		Ok:  ok,
	}
}

func (x *RespMessage) GetSeq() int32 {
	return x.Seq
}
func (x *RespMessage) GetOk() bool {
	return x.Ok
}
