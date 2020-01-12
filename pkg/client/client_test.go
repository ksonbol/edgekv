package client

import (
	"fmt"
	"testing"

	pb "github.com/ksonbol/edgekv/frontend/frontend"
	frmock "github.com/ksonbol/edgekv/frontend/mock_frontend"
	"github.com/ksonbol/edgekv/utils"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
)

type rpcMsg struct {
	msg proto.Message
}

func (r *rpcMsg) Matches(msg interface{}) bool {
	m, ok := msg.(proto.Message)
	if !ok {
		return false
	}
	return proto.Equal(m, r.msg)
}

func (r *rpcMsg) String() string {
	return fmt.Sprintf("is %s", r.msg)
}

type testData struct {
	testKey    string
	testValue  string
	dataType   bool
	testSize   int32
	testStatus int32
}

func newTestData() *testData {
	t := testData{testKey: "test_key", testValue: "test_val", dataType: utils.LocalData, testSize: 10, testStatus: 0}
	return &t
}

func setupTest(t *testing.T) (*gomock.Controller, *frmock.MockFrontendClient, *testData) {
	data := newTestData()
	mockCtrl := gomock.NewController(t)
	mockFrontendClient := frmock.NewMockFrontendClient(mockCtrl)
	return mockCtrl, mockFrontendClient, data
}
func TestGet(t *testing.T) {
	mockCtrl, client, data := setupTest(t)
	defer mockCtrl.Finish()
	req := &pb.GetRequest{Key: data.testKey, Type: data.dataType}
	client.EXPECT().Get(
		gomock.Any(), // expect any value for first parameter
		&rpcMsg{msg: req},
	).Return(&pb.GetResponse{Value: data.testValue, Size: data.testSize}, nil)
	res, err := Get(client, req)
	if err != nil || res.Value != data.testValue || res.Size != data.testSize {
		t.Errorf("Get(%s, %t) = Value: %v, Size: %v, want Value: %v, Size: %v", data.testKey, data.dataType, res.Value,
			res.Size, data.testValue, data.testSize)
	} else {
		t.Log("Get operation succeeded.")
	}
}

func TestPut(t *testing.T) {
	mockCtrl, client, data := setupTest(t)
	defer mockCtrl.Finish()
	req := &pb.PutRequest{Key: data.testKey, Type: data.dataType, Value: data.testValue}
	client.EXPECT().Put(
		gomock.Any(), // expect any value for first parameter
		&rpcMsg{msg: req},
	).Return(&pb.PutResponse{Status: data.testStatus}, nil)
	res, err := Put(client, req)
	if err != nil || res.Status != data.testStatus {
		t.Errorf("Put(%s, %s, %t) = %v, want %v", data.testKey, data.testValue, data.dataType, res.Status, data.testStatus)
	} else {
		t.Log("Put operation succeeded.")
	}
}

func TestDel(t *testing.T) {
	mockCtrl, client, data := setupTest(t)
	defer mockCtrl.Finish()
	req := &pb.DeleteRequest{Key: data.testKey, Type: data.dataType}
	client.EXPECT().Del(
		gomock.Any(), // expect any value for first parameter
		&rpcMsg{msg: req},
	).Return(&pb.DeleteResponse{Status: data.testStatus}, nil)
	res, err := Del(client, req)
	if err != nil || res.Status != data.testStatus {
		t.Errorf("Del(%s, %t) = %v, want %v", data.testKey, data.dataType, res.Status, data.testStatus)
	} else {
		t.Log("Put operation succeeded.")
	}
}
