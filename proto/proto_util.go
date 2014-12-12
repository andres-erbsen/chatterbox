package proto

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
