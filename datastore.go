package datastoreextensions

import (
	"context"
	"errors"
	"reflect"

	"cloud.google.com/go/datastore"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	pb "google.golang.org/genproto/googleapis/datastore/v1"
)

type DatastoreExtensionClient struct {
	client     *datastore.Client
	pbClient   pb.DatastoreClient
	dataset    string // synonim for projectID
	databaseID string // default is "(default)"
}

var ErrNotPbClient = errors.New("client is not a pb.DatastoreClient: Google Datastore library may have been updated, check if Datastore Extensions library has an update too, or create an issue to initiate an update")

func FromClient(client *datastore.Client) (*DatastoreExtensionClient, error) {
	anon := getUnexportedField(reflect.ValueOf(client).Elem().FieldByName("client"))
	pbClient, isPbClient := anon.(pb.DatastoreClient)
	if !isPbClient {
		return nil, ErrNotPbClient
	}

	return &DatastoreExtensionClient{
		client:     client,
		pbClient:   pbClient,
		dataset:    getUnexportedField(reflect.ValueOf(client).Elem().FieldByName("dataset")).(string),
		databaseID: getUnexportedField(reflect.ValueOf(client).Elem().FieldByName("databaseID")).(string),
	}, nil
}

func init() {
	// this init ensures this issue is found during startup of the binary (before first test ever hits this binary), and not on later usage when it's too late
	c, err := dummyClient()
	if err != nil {
		panic(err)
	}
	_, err = FromClient(c)
	if errors.Is(err, ErrNotPbClient) {
		panic(err)
	}
}

func dummyClient() (*datastore.Client, error) {
	return datastore.NewClient(context.Background(), "foobar", option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{})), option.WithEndpoint("never.really.connect.to.fake.datastore.example.com:443"))
}
