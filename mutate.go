package datastoreextensions

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"

	"cloud.google.com/go/datastore"
	pb "google.golang.org/genproto/googleapis/datastore/v1"
	"google.golang.org/grpc"
)

var (
	ErrConflict        = errors.New("conflict")
	ErrMissingRecorder = errors.New("missing recorder in context, make sure to use the same recorder context (WithRecorder) to read from Datastore and for Mutate(WithLocks)")
	ErrMissingEntity   = "missing entity %q in recorder"
)

func (ec *DatastoreExtensionClient) Mutate(ctx context.Context, mutations []*datastore.Mutation) (*pb.CommitResponse, error) {
	return ec.MutateWithLocks(ctx, mutations, nil)
}

func SetBaseVersion(mutation *datastore.Mutation, baseVersion int64) {
	mut, isMut := getUnexportedField(reflect.ValueOf(mutation).Elem().FieldByName("mut")).(*pb.Mutation)
	if !isMut || mut == nil {
		panic("mutation is not a valid pb.Mutation")
	}
	mut.ConflictDetectionStrategy = &pb.Mutation_BaseVersion{BaseVersion: baseVersion}
}

// MutateWithLocks can lock some recently read entities by performing a write-as-read using the BaseVersion stored in the recorder
// This is useful to prevent concurrent modifications to the same entities, while writing other related entities.
// For example, after reading a TaskGroup entity, a Task entity can be modified while the TaskGroup entity can be locked to prevent concurrent modifications.
func (ec *DatastoreExtensionClient) MutateWithLocks(ctx context.Context, muts []*datastore.Mutation, locked []*datastore.Key) (*pb.CommitResponse, error) {
	// collect all locking mutations
	var mutations []*pb.Mutation
	if len(locked) > 0 {
		rec, hasRec := ctx.Value(RecorderKey).(*Recorder)
		if !hasRec || rec == nil {
			return nil, ErrMissingRecorder
		}
		for _, key := range locked {
			lockedEntity, ok := rec.Get(key)
			if !ok {
				return nil, fmt.Errorf(ErrMissingEntity, key.String())
			}
			mutations = append(mutations, &pb.Mutation{
				Operation:                 &pb.Mutation_Update{Update: lockedEntity.Entity},
				ConflictDetectionStrategy: &pb.Mutation_BaseVersion{BaseVersion: lockedEntity.Version},
			})
		}
	}

	// add input datastore mutations to list
	converted, err := mutationProtos(muts)
	if err != nil {
		return nil, err
	}
	mutations = append(mutations, converted...)

	// commit all mutations in a single use transaction (which does not require the additional roundtrips of BeginTransaction/Rollback calls around Commit)
	resp, err := ec.pbClient.Commit(ctx, &pb.CommitRequest{
		ProjectId:  ec.dataset,
		DatabaseId: ec.databaseID,
		Mode:       pb.CommitRequest_TRANSACTIONAL,
		Mutations:  mutations,
		TransactionSelector: &pb.CommitRequest_SingleUseTransaction{SingleUseTransaction: &pb.TransactionOptions{
			Mode: &pb.TransactionOptions_ReadWrite_{ReadWrite: &pb.TransactionOptions_ReadWrite{}},
		}},
	})
	if err != nil {
		return resp, err
	}
	anyConflicts := false
	for _, result := range resp.MutationResults {
		anyConflicts = anyConflicts || result.ConflictDetected
	}
	if anyConflicts {
		return resp, ErrConflict
	}
	return resp, err
}

type RecorderContextKey string

const RecorderKey RecorderContextKey = "recorder"

// NewRecordingInterceptor is LAZY, so you can opt-in to recording when needed, and stores the results in the (request local) context for use within the same logical execution.
func NewRecordingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			return err
		}

		// do nothing if no recorder is attached
		rec, hasRec := ctx.Value(RecorderKey).(*Recorder)
		if !hasRec || rec == nil {
			return nil
		}

		// store Get/GetMulti responses
		commitResp, isCommitResp := reply.(*pb.LookupResponse)
		if isCommitResp {
			rec.AddResults(commitResp.Found)
		}

		// store Run/GetAll responses
		queryResp, isQueryResp := reply.(*pb.RunQueryResponse)
		if isQueryResp {
			rec.AddResults(queryResp.Batch.EntityResults)
		}
		return nil
	}
}

func WithRecorder(ctx context.Context) (context.Context, *Recorder) {
	recorder := &Recorder{}
	return context.WithValue(ctx, RecorderKey, recorder), recorder
}

type Recorder struct {
	Results []*pb.EntityResult
	indexed map[string]*pb.EntityResult
}

func (r *Recorder) AddResults(results []*pb.EntityResult) {
	r.Results = append(r.Results, results...)
	if r.indexed == nil {
		r.indexed = make(map[string]*pb.EntityResult)
	}
	for _, res := range results {
		key, err := protoToKey(res.Entity.Key)
		if err != nil {
			log.Println("error converting key", err) // should never happen for valid responses, but let's be safe and make it diagnosable if it does
			continue
		}
		r.indexed[key.Encode()] = res
	}
}

func (r *Recorder) Get(key *datastore.Key) (*pb.EntityResult, bool) {
	if r.indexed == nil {
		return nil, false
	}
	res, ok := r.indexed[key.Encode()]
	return res, ok
}
