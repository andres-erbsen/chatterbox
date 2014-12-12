package proto

const MAX_MESSAGE_SIZE = 16 * 1024
const SERVER_MESSAGE_SIZE = MAX_MESSAGE_SIZE + 100

func ToProtoByte32List(list [][32]byte) []Byte32 {
	newList := make([]Byte32, 0)
	for _, element := range list {
		newList = append(newList, (Byte32)(element))
	}
	return newList
}

func To32ByteList(list []Byte32) [][32]byte {
	newList := make([][32]byte, 0, 0)
	for _, element := range list {
		newList = append(newList, ([32]byte)(element))
	}
	return newList
}

func Pad(msg []byte, l int) []byte {
	msg = append(msg, 1)
	if l > len(msg) {
		msg = append(msg, make([]byte, l-len(msg))...)
	}
	return msg
}
func Unpad(msg []byte) []byte {
	for len(msg) > 1 && msg[len(msg)-1] == 0 {
		msg = msg[:len(msg)-1]
	}
	if len(msg) > 0 {
		msg = msg[:len(msg)-1]
	}
	return msg
}
