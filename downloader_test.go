package multipartdownloader

import (
	"fmt"
	"testing"
)

func Test1Source (t *testing.T) {
	nConns := []uint{1,2,5,10}
	for _, n := range nConns {
		t.Error(fmt.Sprintf("Failed downloading with a single source and %d connections", n))
	}
}

func Test2Sources (t *testing.T) {
	nConns := []uint{1, 2, 5, 30}
	for _, n := range nConns {
		t.Error(fmt.Sprintf("Failed downloading with 2 sources and %d connections", n))
	}
}

func Test3Sources (t *testing.T) {
	nConns := []uint{1, 2, 3, 5, 25, 26}
	for _, n := range nConns {
		t.Error(fmt.Sprintf("Failed downloading with 3 sources and %d connections", n))
	}
}
