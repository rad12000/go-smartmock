package smartmock_test

import (
	"github.com/rad12000/go-smartmock/integrations/testify/pkg/smartmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"strconv"
	"testing"
)

type TestFuncMock struct {
	mock.Mock
}

func (m *TestFuncMock) Execute(value int) {
	smartmock.Fn1x0(m, m.Execute).Called(value)
}

func TestFunc(t *testing.T) {
	type TestCase struct {
		strVal      string
		val         int
		shouldMatch bool
	}

	testCases := map[string]TestCase{
		"does not match": {
			strVal:      "1",
			val:         2,
			shouldMatch: false,
		},
		"does match": {
			strVal:      "1",
			val:         1,
			shouldMatch: true,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			m := new(TestFuncMock)
			smartmock.Fn1x0(m, m.Execute).
				Match(smartmock.Func(func(val int) bool {
					return strconv.Itoa(val) == tc.strVal
				})).Once()

			if tc.shouldMatch {
				m.Execute(tc.val)
				m.AssertExpectations(t)
			} else {
				defer func() {
					if err := recover(); err != nil {
						assert.Contains(t, err, "Unexpected Method Call")
						return
					}

					t.Errorf("mocked function should not have matched and thefore panicked. But no panic occurred.")
				}()
				m.Execute(tc.val)
			}
		})
	}
}
