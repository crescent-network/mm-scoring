package cmd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	crecmd "github.com/crescent-network/crescent/v3/cmd/crescentd/cmd"

	"github.com/crescent-network/mm-scoring/cmd"
)

type CmdTestSuite struct {
	suite.Suite
}

func TestCmdTestSuite(t *testing.T) {
	suite.Run(t, new(CmdTestSuite))
}

func (suite *CmdTestSuite) SetupTest() {
	crecmd.GetConfig()
}

func (suite *CmdTestSuite) TestMain2() {
	for _, tc := range []struct {
		dir         string
		startHeight int64
		endHeight   int64
	}{
		{
			startHeight: 478559,
		},
	} {
		suite.Run(tc.dir, func() {
			res := cmd.Main(tc.startHeight)
			fmt.Println(res)
		})
	}
}
