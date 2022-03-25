package test

import (
	"github.com/stretchr/testify/mock"
)

type MockAuthDataSource struct {
	mock.Mock
}

func (m *MockAuthDataSource) Header(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *MockAuthDataSource) Cookie(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *MockAuthDataSource) Query(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *MockAuthDataSource) Form(name string) string {
	args := m.Called(name)
	return args.String(0)
}