package ratchet

import (
	"time"

	protobuf "code.google.com/p/gogoprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/proto"
	. "github.com/andres-erbsen/chatterbox/ratchet/proto"
)

func (r *Ratchet) Proto() protobuf.Message              { return NewRatchetStateFromFace(r) }
func (r *Ratchet) GetRootKey() *proto.Byte32            { return (*proto.Byte32)(&r.rootKey) }
func (r *Ratchet) GetOurRatchetPrivate() *proto.Byte32  { return (*proto.Byte32)(&r.ourRatchetPrivate) }
func (r *Ratchet) GetTheirRatchetPublic() *proto.Byte32 { return (*proto.Byte32)(&r.theirRatchetPublic) }
func (r *Ratchet) GetRatchet() bool                     { return r.ratchet }
func (r *Ratchet) GetSendHeaderKey() *proto.Byte32      { return (*proto.Byte32)(&r.sendHeaderKey) }
func (r *Ratchet) GetRecvHeaderKey() *proto.Byte32      { return (*proto.Byte32)(&r.recvHeaderKey) }
func (r *Ratchet) GetNextSendHeaderKey() *proto.Byte32  { return (*proto.Byte32)(&r.nextSendHeaderKey) }
func (r *Ratchet) GetNextRecvHeaderKey() *proto.Byte32  { return (*proto.Byte32)(&r.nextRecvHeaderKey) }
func (r *Ratchet) GetSendChainKey() *proto.Byte32       { return (*proto.Byte32)(&r.sendChainKey) }
func (r *Ratchet) GetRecvChainKey() *proto.Byte32       { return (*proto.Byte32)(&r.recvChainKey) }
func (r *Ratchet) GetSendCount() uint32                 { return r.sendCount }
func (r *Ratchet) GetRecvCount() uint32                 { return r.recvCount }
func (r *Ratchet) GetPrevSendCount() uint32             { return r.prevSendCount }

func (r *Ratchet) GetSavedKeys() []RatchetState_SavedKeys {
	ret := make([]RatchetState_SavedKeys, len(r.saved))
	i := 0
	for headerKey, messageKeys := range r.saved {
		ret[i].HeaderKey = (*proto.Byte32)(&headerKey)
		ret[i].MessageKeys = make([]RatchetState_SavedKeys_MessageKey, len(messageKeys))
		j := 0
		for messageNum, savedKey := range messageKeys {
			ret[i].MessageKeys[j].Num = messageNum
			ret[i].MessageKeys[j].Key = (*proto.Byte32)(&savedKey.key)
			ret[i].MessageKeys[j].CreationTime = savedKey.timestamp.Unix()
			j++
		}
		i++
	}
	return ret
}

func (r *Ratchet) Marshal() ([]byte, error)          { return NewRatchetStateFromFace(r).Marshal() }
func (r *Ratchet) MarshalTo(out []byte) (int, error) { return NewRatchetStateFromFace(r).MarshalTo(out) }

func (r *Ratchet) Unmarshal(data []byte) error {
	rs := new(RatchetState)
	err := rs.Unmarshal(data)
	r.FillFromFace(rs)
	return err
}

func (r *Ratchet) FillFromFace(that RatchetStateFace) *Ratchet {
	r.rootKey = *that.GetRootKey()
	r.sendHeaderKey = *that.GetSendHeaderKey()
	r.recvHeaderKey = *that.GetRecvHeaderKey()
	r.nextSendHeaderKey = *that.GetNextSendHeaderKey()
	r.nextRecvHeaderKey = *that.GetNextRecvHeaderKey()
	r.sendChainKey = *that.GetSendChainKey()
	r.recvChainKey = *that.GetRecvChainKey()
	r.sendCount = that.GetSendCount()
	r.recvCount = that.GetRecvCount()
	r.prevSendCount = that.GetPrevSendCount()
	r.ratchet = that.GetRatchet()
	r.ourRatchetPrivate = *that.GetOurRatchetPrivate()
	r.theirRatchetPublic = *that.GetTheirRatchetPublic()
	r.saved = make(map[[32]byte]map[uint32]savedKey)
	for _, saved := range that.GetSavedKeys() {
		messageKeys := make(map[uint32]savedKey)
		for _, messageKey := range saved.MessageKeys {
			messageKeys[messageKey.GetNum()] = savedKey{
				key:       *messageKey.Key,
				timestamp: time.Unix(messageKey.GetCreationTime(), 0),
			}
		}
		r.saved[*saved.HeaderKey] = messageKeys
	}
	return r
}

func NewRatchetFromFace(that RatchetStateFace) *Ratchet {
	return new(Ratchet).FillFromFace(that)
}
