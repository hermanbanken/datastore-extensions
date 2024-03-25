package datastoreextensions

import (
	"reflect"
	"unsafe"

	"cloud.google.com/go/datastore"
	pb "google.golang.org/genproto/googleapis/datastore/v1"
)

// getUnexportedField
// https://stackoverflow.com/questions/42664837/how-to-access-unexported-struct-fields
func getUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

//go:linkname mutationProtos cloud.google.com/go/datastore.mutationProtos
func mutationProtos(muts []*datastore.Mutation) ([]*pb.Mutation, error)

//go:linkname keyToProto cloud.google.com/go/datastore.keyToProto
func keyToProto(p *datastore.Key) *pb.Key

//go:linkname protoToKey cloud.google.com/go/datastore.protoToKey
func protoToKey(p *pb.Key) (*datastore.Key, error)

func ProtoToKey(p *pb.Key) (*datastore.Key, error) {
	return protoToKey(p)
}
