package serverpool

import (
	"load-balancer/utils"
	"testing"

	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	utils.Logger = zap.NewNop()

	m.Run()
}
