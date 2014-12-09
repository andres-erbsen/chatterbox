// Code generated by protoc-gen-gogo.
// source: LocalConversationMetadata.proto
// DO NOT EDIT!

package proto

import testing16 "testing"
import math_rand16 "math/rand"
import time16 "time"
import github_com_gogo_protobuf_proto12 "github.com/gogo/protobuf/proto"
import testing17 "testing"
import math_rand17 "math/rand"
import time17 "time"
import encoding_json4 "encoding/json"
import testing18 "testing"
import math_rand18 "math/rand"
import time18 "time"
import github_com_gogo_protobuf_proto13 "github.com/gogo/protobuf/proto"
import math_rand19 "math/rand"
import time19 "time"
import testing19 "testing"
import github_com_gogo_protobuf_proto14 "github.com/gogo/protobuf/proto"

func TestConversationMetadataProto(t *testing16.T) {
	popr := math_rand16.New(math_rand16.NewSource(time16.Now().UnixNano()))
	p := NewPopulatedConversationMetadata(popr, false)
	data, err := github_com_gogo_protobuf_proto12.Marshal(p)
	if err != nil {
		panic(err)
	}
	msg := &ConversationMetadata{}
	if err := github_com_gogo_protobuf_proto12.Unmarshal(data, msg); err != nil {
		panic(err)
	}
	for i := range data {
		data[i] = byte(popr.Intn(256))
	}
	if !p.Equal(msg) {
		t.Fatalf("%#v !Proto %#v", msg, p)
	}
}

func TestConversationMetadataMarshalTo(t *testing16.T) {
	popr := math_rand16.New(math_rand16.NewSource(time16.Now().UnixNano()))
	p := NewPopulatedConversationMetadata(popr, false)
	size := p.Size()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(popr.Intn(256))
	}
	_, err := p.MarshalTo(data)
	if err != nil {
		panic(err)
	}
	msg := &ConversationMetadata{}
	if err := github_com_gogo_protobuf_proto12.Unmarshal(data, msg); err != nil {
		panic(err)
	}
	for i := range data {
		data[i] = byte(popr.Intn(256))
	}
	if !p.Equal(msg) {
		t.Fatalf("%#v !Proto %#v", msg, p)
	}
}

func BenchmarkConversationMetadataProtoMarshal(b *testing16.B) {
	popr := math_rand16.New(math_rand16.NewSource(616))
	total := 0
	pops := make([]*ConversationMetadata, 10000)
	for i := 0; i < 10000; i++ {
		pops[i] = NewPopulatedConversationMetadata(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := github_com_gogo_protobuf_proto12.Marshal(pops[i%10000])
		if err != nil {
			panic(err)
		}
		total += len(data)
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkConversationMetadataProtoUnmarshal(b *testing16.B) {
	popr := math_rand16.New(math_rand16.NewSource(616))
	total := 0
	datas := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		data, err := github_com_gogo_protobuf_proto12.Marshal(NewPopulatedConversationMetadata(popr, false))
		if err != nil {
			panic(err)
		}
		datas[i] = data
	}
	msg := &ConversationMetadata{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += len(datas[i%10000])
		if err := github_com_gogo_protobuf_proto12.Unmarshal(datas[i%10000], msg); err != nil {
			panic(err)
		}
	}
	b.SetBytes(int64(total / b.N))
}

func TestConversationMetadataJSON(t *testing17.T) {
	popr := math_rand17.New(math_rand17.NewSource(time17.Now().UnixNano()))
	p := NewPopulatedConversationMetadata(popr, true)
	jsondata, err := encoding_json4.Marshal(p)
	if err != nil {
		panic(err)
	}
	msg := &ConversationMetadata{}
	err = encoding_json4.Unmarshal(jsondata, msg)
	if err != nil {
		panic(err)
	}
	if !p.Equal(msg) {
		t.Fatalf("%#v !Json Equal %#v", msg, p)
	}
}
func TestConversationMetadataProtoText(t *testing18.T) {
	popr := math_rand18.New(math_rand18.NewSource(time18.Now().UnixNano()))
	p := NewPopulatedConversationMetadata(popr, true)
	data := github_com_gogo_protobuf_proto13.MarshalTextString(p)
	msg := &ConversationMetadata{}
	if err := github_com_gogo_protobuf_proto13.UnmarshalText(data, msg); err != nil {
		panic(err)
	}
	if !p.Equal(msg) {
		t.Fatalf("%#v !Proto %#v", msg, p)
	}
}

func TestConversationMetadataProtoCompactText(t *testing18.T) {
	popr := math_rand18.New(math_rand18.NewSource(time18.Now().UnixNano()))
	p := NewPopulatedConversationMetadata(popr, true)
	data := github_com_gogo_protobuf_proto13.CompactTextString(p)
	msg := &ConversationMetadata{}
	if err := github_com_gogo_protobuf_proto13.UnmarshalText(data, msg); err != nil {
		panic(err)
	}
	if !p.Equal(msg) {
		t.Fatalf("%#v !Proto %#v", msg, p)
	}
}

func TestConversationMetadataSize(t *testing19.T) {
	popr := math_rand19.New(math_rand19.NewSource(time19.Now().UnixNano()))
	p := NewPopulatedConversationMetadata(popr, true)
	size2 := github_com_gogo_protobuf_proto14.Size(p)
	data, err := github_com_gogo_protobuf_proto14.Marshal(p)
	if err != nil {
		panic(err)
	}
	size := p.Size()
	if len(data) != size {
		t.Fatalf("size %v != marshalled size %v", size, len(data))
	}
	if size2 != size {
		t.Fatalf("size %v != before marshal proto.Size %v", size, size2)
	}
	size3 := github_com_gogo_protobuf_proto14.Size(p)
	if size3 != size {
		t.Fatalf("size %v != after marshal proto.Size %v", size, size3)
	}
}

func BenchmarkConversationMetadataSize(b *testing19.B) {
	popr := math_rand19.New(math_rand19.NewSource(616))
	total := 0
	pops := make([]*ConversationMetadata, 1000)
	for i := 0; i < 1000; i++ {
		pops[i] = NewPopulatedConversationMetadata(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += pops[i%1000].Size()
	}
	b.SetBytes(int64(total / b.N))
}

//These tests are generated by github.com/gogo/protobuf/plugin/testgen
