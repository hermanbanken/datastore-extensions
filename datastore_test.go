package datastoreextensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromClient(t *testing.T) {
	c, err := dummyClient()
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	ec, err := FromClient(c)
	if assert.NoError(t, err) && assert.NotNil(t, ec) {
		assert.NotNil(t, ec.client)
		assert.NotNil(t, ec.pbClient)
	}
}
