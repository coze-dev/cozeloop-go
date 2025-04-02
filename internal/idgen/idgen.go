// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package idgen

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	once        sync.Once
	idGenerator IDGenerator
)

type IDGenerator interface {
	GenId() uint64
}

type multiDeltaIdGenerator struct {
	idGenerators []IDGenerator
	index        uint64
	num          uint64
}

func GetMultipleDeltaIdGenerator() IDGenerator {
	once.Do(func() {
		idGenerator = newMultipleDeltaIdGenerator(math.MaxInt64, 1, 10)
	})
	return idGenerator
}

func newMultipleDeltaIdGenerator(reseedThreshold uint64, delta uint64, num uint64) IDGenerator {
	var idGenerators []IDGenerator
	for i := 0; i < int(num); i++ {
		idGenerators = append(idGenerators, newAccumulateIdGenerator(reseedThreshold, delta))
	}
	return &multiDeltaIdGenerator{
		idGenerators: idGenerators,
		index:        0,
		num:          num,
	}
}

func (m *multiDeltaIdGenerator) GenId() uint64 {
	return m.idGenerators[m.getIndex()].GenId()
}

func (m *multiDeltaIdGenerator) getIndex() uint64 {
	id := atomic.AddUint64(&m.index, 1)
	return id % m.num
}

func newAccumulateIdGenerator(reseedThreshold uint64, delta uint64) IDGenerator {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	randFunc := func() uint64 {
		return uint64(r.Int63n(int64(minUnit64(math.MaxInt64, reseedThreshold))))
	}
	return &deltaIdGenerator{
		randomNumber: randFunc,
		seed:         randFunc(),
		maxId:        reseedThreshold,
		delta:        delta,
	}
}

// Generate IDs incrementally.
type deltaIdGenerator struct {
	randomNumber func() uint64
	seed         uint64
	maxId        uint64
	delta        uint64
}

func (t *deltaIdGenerator) GenId() uint64 {
	id := t.addAndGet()
	if id >= t.maxId {
		t.resetSeed()
		return t.GenId()
	}
	return id
}

func (t *deltaIdGenerator) addAndGet() uint64 {
	return atomic.AddUint64(&t.seed, t.delta)
}

func (t *deltaIdGenerator) resetSeed() {
	atomic.StoreUint64(&t.seed, t.randomNumber())
}

func minUnit64(v0, v1 uint64) uint64 {
	if v0 > v1 {
		return v1
	}
	return v0
}
