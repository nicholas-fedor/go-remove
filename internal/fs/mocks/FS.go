// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"github.com/nicholas-fedor/go-remove/internal/logger"
	mock "github.com/stretchr/testify/mock"
)

// NewMockFS creates a new instance of MockFS. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockFS(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockFS {
	mock := &MockFS{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockFS is an autogenerated mock type for the FS type
type MockFS struct {
	mock.Mock
}

type MockFS_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFS) EXPECT() *MockFS_Expecter {
	return &MockFS_Expecter{mock: &_m.Mock}
}

// AdjustBinaryPath provides a mock function for the type MockFS
func (_mock *MockFS) AdjustBinaryPath(dir string, binary string) string {
	ret := _mock.Called(dir, binary)

	if len(ret) == 0 {
		panic("no return value specified for AdjustBinaryPath")
	}

	var r0 string
	if returnFunc, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = returnFunc(dir, binary)
	} else {
		r0 = ret.Get(0).(string)
	}
	return r0
}

// MockFS_AdjustBinaryPath_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AdjustBinaryPath'
type MockFS_AdjustBinaryPath_Call struct {
	*mock.Call
}

// AdjustBinaryPath is a helper method to define mock.On call
//   - dir string
//   - binary string
func (_e *MockFS_Expecter) AdjustBinaryPath(dir interface{}, binary interface{}) *MockFS_AdjustBinaryPath_Call {
	return &MockFS_AdjustBinaryPath_Call{Call: _e.mock.On("AdjustBinaryPath", dir, binary)}
}

func (_c *MockFS_AdjustBinaryPath_Call) Run(run func(dir string, binary string)) *MockFS_AdjustBinaryPath_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		var arg1 string
		if args[1] != nil {
			arg1 = args[1].(string)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockFS_AdjustBinaryPath_Call) Return(s string) *MockFS_AdjustBinaryPath_Call {
	_c.Call.Return(s)
	return _c
}

func (_c *MockFS_AdjustBinaryPath_Call) RunAndReturn(run func(dir string, binary string) string) *MockFS_AdjustBinaryPath_Call {
	_c.Call.Return(run)
	return _c
}

// DetermineBinDir provides a mock function for the type MockFS
func (_mock *MockFS) DetermineBinDir(useGoroot bool) (string, error) {
	ret := _mock.Called(useGoroot)

	if len(ret) == 0 {
		panic("no return value specified for DetermineBinDir")
	}

	var r0 string
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(bool) (string, error)); ok {
		return returnFunc(useGoroot)
	}
	if returnFunc, ok := ret.Get(0).(func(bool) string); ok {
		r0 = returnFunc(useGoroot)
	} else {
		r0 = ret.Get(0).(string)
	}
	if returnFunc, ok := ret.Get(1).(func(bool) error); ok {
		r1 = returnFunc(useGoroot)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockFS_DetermineBinDir_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DetermineBinDir'
type MockFS_DetermineBinDir_Call struct {
	*mock.Call
}

// DetermineBinDir is a helper method to define mock.On call
//   - useGoroot bool
func (_e *MockFS_Expecter) DetermineBinDir(useGoroot interface{}) *MockFS_DetermineBinDir_Call {
	return &MockFS_DetermineBinDir_Call{Call: _e.mock.On("DetermineBinDir", useGoroot)}
}

func (_c *MockFS_DetermineBinDir_Call) Run(run func(useGoroot bool)) *MockFS_DetermineBinDir_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 bool
		if args[0] != nil {
			arg0 = args[0].(bool)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *MockFS_DetermineBinDir_Call) Return(s string, err error) *MockFS_DetermineBinDir_Call {
	_c.Call.Return(s, err)
	return _c
}

func (_c *MockFS_DetermineBinDir_Call) RunAndReturn(run func(useGoroot bool) (string, error)) *MockFS_DetermineBinDir_Call {
	_c.Call.Return(run)
	return _c
}

// ListBinaries provides a mock function for the type MockFS
func (_mock *MockFS) ListBinaries(dir string) []string {
	ret := _mock.Called(dir)

	if len(ret) == 0 {
		panic("no return value specified for ListBinaries")
	}

	var r0 []string
	if returnFunc, ok := ret.Get(0).(func(string) []string); ok {
		r0 = returnFunc(dir)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}
	return r0
}

// MockFS_ListBinaries_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListBinaries'
type MockFS_ListBinaries_Call struct {
	*mock.Call
}

// ListBinaries is a helper method to define mock.On call
//   - dir string
func (_e *MockFS_Expecter) ListBinaries(dir interface{}) *MockFS_ListBinaries_Call {
	return &MockFS_ListBinaries_Call{Call: _e.mock.On("ListBinaries", dir)}
}

func (_c *MockFS_ListBinaries_Call) Run(run func(dir string)) *MockFS_ListBinaries_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *MockFS_ListBinaries_Call) Return(strings []string) *MockFS_ListBinaries_Call {
	_c.Call.Return(strings)
	return _c
}

func (_c *MockFS_ListBinaries_Call) RunAndReturn(run func(dir string) []string) *MockFS_ListBinaries_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveBinary provides a mock function for the type MockFS
func (_mock *MockFS) RemoveBinary(binaryPath string, name string, verbose bool, logger1 logger.Logger) error {
	ret := _mock.Called(binaryPath, name, verbose, logger1)

	if len(ret) == 0 {
		panic("no return value specified for RemoveBinary")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(string, string, bool, logger.Logger) error); ok {
		r0 = returnFunc(binaryPath, name, verbose, logger1)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockFS_RemoveBinary_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveBinary'
type MockFS_RemoveBinary_Call struct {
	*mock.Call
}

// RemoveBinary is a helper method to define mock.On call
//   - binaryPath string
//   - name string
//   - verbose bool
//   - logger1 logger.Logger
func (_e *MockFS_Expecter) RemoveBinary(binaryPath interface{}, name interface{}, verbose interface{}, logger1 interface{}) *MockFS_RemoveBinary_Call {
	return &MockFS_RemoveBinary_Call{Call: _e.mock.On("RemoveBinary", binaryPath, name, verbose, logger1)}
}

func (_c *MockFS_RemoveBinary_Call) Run(run func(binaryPath string, name string, verbose bool, logger1 logger.Logger)) *MockFS_RemoveBinary_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 string
		if args[0] != nil {
			arg0 = args[0].(string)
		}
		var arg1 string
		if args[1] != nil {
			arg1 = args[1].(string)
		}
		var arg2 bool
		if args[2] != nil {
			arg2 = args[2].(bool)
		}
		var arg3 logger.Logger
		if args[3] != nil {
			arg3 = args[3].(logger.Logger)
		}
		run(
			arg0,
			arg1,
			arg2,
			arg3,
		)
	})
	return _c
}

func (_c *MockFS_RemoveBinary_Call) Return(err error) *MockFS_RemoveBinary_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockFS_RemoveBinary_Call) RunAndReturn(run func(binaryPath string, name string, verbose bool, logger1 logger.Logger) error) *MockFS_RemoveBinary_Call {
	_c.Call.Return(run)
	return _c
}
