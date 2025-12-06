package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetDigitArray(t *testing.T) {
	assert.Equal(t, GetDigitArray(0), []int64{0})
	assert.Equal(t, GetDigitArray(1), []int64{1})
	assert.Equal(t, GetDigitArray(725), []int64{7, 2, 5})
	assert.Equal(t, GetDigitArray(123456), []int64{1, 2, 3, 4, 5, 6})
}
