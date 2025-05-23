// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go
//
// Generated by this command:
//
//	mockgen -source interface.go -destination interface_mock.go -package finalizer
//

// Package finalizer is a generated GoMock package.
package finalizer

import (
	context "context"
	reflect "reflect"

	entity "github.com/n-r-w/collector/internal/entity"
	gomock "go.uber.org/mock/gomock"
)

// MockICollectionReader is a mock of ICollectionReader interface.
type MockICollectionReader struct {
	ctrl     *gomock.Controller
	recorder *MockICollectionReaderMockRecorder
}

// MockICollectionReaderMockRecorder is the mock recorder for MockICollectionReader.
type MockICollectionReaderMockRecorder struct {
	mock *MockICollectionReader
}

// NewMockICollectionReader creates a new mock instance.
func NewMockICollectionReader(ctrl *gomock.Controller) *MockICollectionReader {
	mock := &MockICollectionReader{ctrl: ctrl}
	mock.recorder = &MockICollectionReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockICollectionReader) EXPECT() *MockICollectionReaderMockRecorder {
	return m.recorder
}

// GetCollections mocks base method.
func (m *MockICollectionReader) GetCollections(ctx context.Context, filter entity.CollectionFilter) ([]entity.Collection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCollections", ctx, filter)
	ret0, _ := ret[0].([]entity.Collection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCollections indicates an expected call of GetCollections.
func (mr *MockICollectionReaderMockRecorder) GetCollections(ctx, filter any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCollections", reflect.TypeOf((*MockICollectionReader)(nil).GetCollections), ctx, filter)
}

// MockIStatusChanger is a mock of IStatusChanger interface.
type MockIStatusChanger struct {
	ctrl     *gomock.Controller
	recorder *MockIStatusChangerMockRecorder
}

// MockIStatusChangerMockRecorder is the mock recorder for MockIStatusChanger.
type MockIStatusChangerMockRecorder struct {
	mock *MockIStatusChanger
}

// NewMockIStatusChanger creates a new mock instance.
func NewMockIStatusChanger(ctrl *gomock.Controller) *MockIStatusChanger {
	mock := &MockIStatusChanger{ctrl: ctrl}
	mock.recorder = &MockIStatusChangerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIStatusChanger) EXPECT() *MockIStatusChangerMockRecorder {
	return m.recorder
}

// UpdateStatus mocks base method.
func (m *MockIStatusChanger) UpdateStatus(ctx context.Context, collectionID entity.CollectionID, status entity.CollectionStatus) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", ctx, collectionID, status)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockIStatusChangerMockRecorder) UpdateStatus(ctx, collectionID, status any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockIStatusChanger)(nil).UpdateStatus), ctx, collectionID, status)
}

// MockIResultChanGetter is a mock of IResultChanGetter interface.
type MockIResultChanGetter struct {
	ctrl     *gomock.Controller
	recorder *MockIResultChanGetterMockRecorder
}

// MockIResultChanGetterMockRecorder is the mock recorder for MockIResultChanGetter.
type MockIResultChanGetterMockRecorder struct {
	mock *MockIResultChanGetter
}

// NewMockIResultChanGetter creates a new mock instance.
func NewMockIResultChanGetter(ctrl *gomock.Controller) *MockIResultChanGetter {
	mock := &MockIResultChanGetter{ctrl: ctrl}
	mock.recorder = &MockIResultChanGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIResultChanGetter) EXPECT() *MockIResultChanGetterMockRecorder {
	return m.recorder
}

// GetResultChan mocks base method.
func (m *MockIResultChanGetter) GetResultChan(ctx context.Context, collectionID entity.CollectionID, limit int) (<-chan entity.RequestChunk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResultChan", ctx, collectionID, limit)
	ret0, _ := ret[0].(<-chan entity.RequestChunk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetResultChan indicates an expected call of GetResultChan.
func (mr *MockIResultChanGetterMockRecorder) GetResultChan(ctx, collectionID, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResultChan", reflect.TypeOf((*MockIResultChanGetter)(nil).GetResultChan), ctx, collectionID, limit)
}

// MockIResultChanSaver is a mock of IResultChanSaver interface.
type MockIResultChanSaver struct {
	ctrl     *gomock.Controller
	recorder *MockIResultChanSaverMockRecorder
}

// MockIResultChanSaverMockRecorder is the mock recorder for MockIResultChanSaver.
type MockIResultChanSaverMockRecorder struct {
	mock *MockIResultChanSaver
}

// NewMockIResultChanSaver creates a new mock instance.
func NewMockIResultChanSaver(ctrl *gomock.Controller) *MockIResultChanSaver {
	mock := &MockIResultChanSaver{ctrl: ctrl}
	mock.recorder = &MockIResultChanSaverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIResultChanSaver) EXPECT() *MockIResultChanSaverMockRecorder {
	return m.recorder
}

// SaveResultChan mocks base method.
func (m *MockIResultChanSaver) SaveResultChan(ctx context.Context, collectionID entity.CollectionID, requests <-chan entity.RequestChunk) (entity.ResultID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveResultChan", ctx, collectionID, requests)
	ret0, _ := ret[0].(entity.ResultID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SaveResultChan indicates an expected call of SaveResultChan.
func (mr *MockIResultChanSaverMockRecorder) SaveResultChan(ctx, collectionID, requests any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveResultChan", reflect.TypeOf((*MockIResultChanSaver)(nil).SaveResultChan), ctx, collectionID, requests)
}

// MockICollectionResultUpdater is a mock of ICollectionResultUpdater interface.
type MockICollectionResultUpdater struct {
	ctrl     *gomock.Controller
	recorder *MockICollectionResultUpdaterMockRecorder
}

// MockICollectionResultUpdaterMockRecorder is the mock recorder for MockICollectionResultUpdater.
type MockICollectionResultUpdaterMockRecorder struct {
	mock *MockICollectionResultUpdater
}

// NewMockICollectionResultUpdater creates a new mock instance.
func NewMockICollectionResultUpdater(ctrl *gomock.Controller) *MockICollectionResultUpdater {
	mock := &MockICollectionResultUpdater{ctrl: ctrl}
	mock.recorder = &MockICollectionResultUpdaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockICollectionResultUpdater) EXPECT() *MockICollectionResultUpdaterMockRecorder {
	return m.recorder
}

// UpdateResultID mocks base method.
func (m *MockICollectionResultUpdater) UpdateResultID(ctx context.Context, collectionID entity.CollectionID, resultID entity.ResultID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateResultID", ctx, collectionID, resultID)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateResultID indicates an expected call of UpdateResultID.
func (mr *MockICollectionResultUpdaterMockRecorder) UpdateResultID(ctx, collectionID, resultID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateResultID", reflect.TypeOf((*MockICollectionResultUpdater)(nil).UpdateResultID), ctx, collectionID, resultID)
}

// MockILocker is a mock of ILocker interface.
type MockILocker struct {
	ctrl     *gomock.Controller
	recorder *MockILockerMockRecorder
}

// MockILockerMockRecorder is the mock recorder for MockILocker.
type MockILockerMockRecorder struct {
	mock *MockILocker
}

// NewMockILocker creates a new mock instance.
func NewMockILocker(ctrl *gomock.Controller) *MockILocker {
	mock := &MockILocker{ctrl: ctrl}
	mock.recorder = &MockILockerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockILocker) EXPECT() *MockILockerMockRecorder {
	return m.recorder
}

// TryLockFunc mocks base method.
func (m *MockILocker) TryLockFunc(ctx context.Context, key entity.LockKey, fn func(context.Context) error) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TryLockFunc", ctx, key, fn)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TryLockFunc indicates an expected call of TryLockFunc.
func (mr *MockILockerMockRecorder) TryLockFunc(ctx, key, fn any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TryLockFunc", reflect.TypeOf((*MockILocker)(nil).TryLockFunc), ctx, key, fn)
}
