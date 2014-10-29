package proto

import "errors"

type Byte32 [32]byte

func (b Byte32) Size() int {
	return 32
}

func (b Byte32) Marshal() ([]byte, error) {
	buffer := make([]byte, 32)
	b.MarshalTo(buffer)
	return buffer, nil
}

func (b Byte32) MarshalTo(data []byte) (int, error) {
	if len(data) < 32 {
		return 0, errors.New("Marshal Byte32: buf too small")
	}
	copy(data, b[:])
	return 32, nil
}

func (b *Byte32) Unmarshal(data []byte) error {
	if len(data) != 32 {
		return errors.New("Unmarshal Byte32: != 32 bytes")
	}
	copy(b[:], data)
	return nil
}

type randy interface {
	Intn(n int) int
}

func NewPopulatedByte32(r randy) *Byte32 {
	ret := new([32]byte)
	for i := 0; i < 32; i++ {
		ret[i] = byte(r.Intn(255))
	}
	return (*Byte32)(ret)
}

func (this *Byte32) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}
	that1, ok := that.(Byte32)
	if !ok {
		return false
	}
	return *this == that1
}
