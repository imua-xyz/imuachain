package oracle

import "time"

func (s *XChainTestSuite) moveToAndCheck(height int64) {
	_, err := s.network.WaitForStateHeightWithTimeout(height, 120*time.Second)
	s.Require().NoError(err)
}

func (s *XChainTestSuite) moveNAndCheck(n int64) {
	for i := int64(0); i < n; i++ {
		err := s.network.WaitForStateNextBlock()
		s.Require().NoError(err)
	}
}
