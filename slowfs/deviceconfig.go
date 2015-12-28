// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slowfs

import (
	"fmt"
	"strings"
	"time"
)

// FsyncStrategy indicates which strategy to use for fsync simulation.
type FsyncStrategy int

const (
	// NoFsync indicates a strategy where fsync takes zero time.
	NoFsync FsyncStrategy = iota
	// DumbFsync indicates a strategy where fsync takes ten seek times (chosen arbitrarily).
	DumbFsync
	// WriteBackCachedFsync indicates a simulation of write back cache. This means writes will take
	// very little time, and writing back that data to disk will be simulated to happen during spare
	// IO time. When fsync is called on a file, how much unwritten data remaining for that file
	// determines how long the fsync takes.
	WriteBackCachedFsync
)

func (f FsyncStrategy) String() string {
	switch f {
	case NoFsync:
		return "NoFsync"
	case DumbFsync:
		return "DumbFsync"
	case WriteBackCachedFsync:
		return "WriteBackCachedFsync"
	default:
		return "unknown fsync strategy"
	}
}

// ParseFsyncStrategyFromString parses a FsyncStrategy from a string. There can be multiple ways to
// specify each FsyncStrategy (e.g. nofsync, none, and no all mean 'NoFsync'). This function is
// case insensitive.
func ParseFsyncStrategyFromString(s string) (FsyncStrategy, error) {
	switch strings.ToLower(s) {
	case "nofsync", "none", "no":
		return NoFsync, nil
	case "dumbfsync", "dumb":
		return DumbFsync, nil
	case "writebackcachedfsync", "writebackcache", "wbc":
		return WriteBackCachedFsync, nil
	default:
		return 0, fmt.Errorf("unknown fsync strategy %s", s)
	}
}

// WriteStrategy indicates which strategy to use for write simulation.
type WriteStrategy int

const (
	// FastWrite means writes will take zero time, as if they were cached.
	// Useful in conjunction with WriteBackCachedFsync
	FastWrite WriteStrategy = iota
	// SimulateWrite means writes will act in the same way as reads -- that is,
	// seeking if non-sequential, and being written out at the speed specified
	// by WriteBytesPerSecond.
	SimulateWrite
)

func (w WriteStrategy) String() string {
	switch w {
	case FastWrite:
		return "FastWrite"
	case SimulateWrite:
		return "SimulateWrite"
	default:
		return "unknown write strategy"
	}
}

// ParseWriteStrategyFromString parses a WriteStrategy from the given string. This function is
// case insensitive, and also accepts synonyms for each WriteStrategy. For example, fastwrite and
// fast both map to the FastWrite strategy.
func ParseWriteStrategyFromString(s string) (WriteStrategy, error) {
	switch strings.ToLower(s) {
	case "fastwrite", "fast":
		return FastWrite, nil
	case "simulatewrite", "simulate":
		return SimulateWrite, nil
	default:
		return 0, fmt.Errorf("unknown write strategy %s", s)
	}
}

// DeviceConfig is used to describe how a physical medium acts (e.g. rotational hard drive).
type DeviceConfig struct {
	// SeekWindow describes how many bytes ahead in a file we can access before considering
	// it a seek.
	SeekWindow NumBytes

	// SeekTime denotes the average time of a seek.
	SeekTime time.Duration

	// ReadBytesPerSecond denotes how many bytes we can read per second.
	ReadBytesPerSecond NumBytes

	// ReadBytesPerSecond denotes how many bytes we can write per second.
	WriteBytesPerSecond NumBytes

	// AllocateBytesPerSecond denotes how many bytes we can allocate using
	// fallocate per second.
	AllocateBytesPerSecond NumBytes

	// RequestReorderMaxDelay denotes how much later a request can be by timestamp after a previous
	// one and still be reordered before it.
	RequestReorderMaxDelay time.Duration

	// FsyncStrategy denotes which algorithm to use for modeling fsync.
	FsyncStrategy FsyncStrategy

	// WriteStrategy denotes which algorithm to use for modeling writes.
	WriteStrategy WriteStrategy

	// MetadataOpTime denotes how long metadata operations (like chmod, chown, etc) should take.
	MetadataOpTime time.Duration
}

// WriteTime computes how long writing numBytes will take.
func (dc *DeviceConfig) WriteTime(numBytes NumBytes) time.Duration {
	return computeTimeFromThroughput(numBytes, dc.WriteBytesPerSecond)
}

// ReadTime computes how long reading numBytes will take.
func (dc *DeviceConfig) ReadTime(numBytes NumBytes) time.Duration {
	return computeTimeFromThroughput(numBytes, dc.ReadBytesPerSecond)
}

// AllocateTime computes how long allocating numBytes will take.
func (dc *DeviceConfig) AllocateTime(numBytes NumBytes) time.Duration {
	return computeTimeFromThroughput(numBytes, dc.AllocateBytesPerSecond)
}

// WritableBytes computes how many bytes can be written in the given duration.
func (dc *DeviceConfig) WritableBytes(duration time.Duration) NumBytes {
	return computeBytesFromTime(duration, dc.WriteBytesPerSecond)
}

// ReadableBytes computes how many bytes can be read in the given duration.
func (dc *DeviceConfig) ReadableBytes(duration time.Duration) NumBytes {
	return computeBytesFromTime(duration, dc.ReadBytesPerSecond)
}

func computeTimeFromThroughput(numBytes, bytesPerSecond NumBytes) time.Duration {
	return time.Duration(float64(numBytes) / float64(bytesPerSecond) * float64(time.Second))
}

func computeBytesFromTime(duration time.Duration, bytesPerSecond NumBytes) NumBytes {
	if duration <= 0 {
		return 0
	}
	return NumBytes(float64(duration) / float64(time.Second) * float64(bytesPerSecond))
}

// HardDriveDeviceConfig is a basic model of a 7200rpm hard disk.
var HardDriveDeviceConfig = DeviceConfig{
	SeekWindow:          4 * Kibibyte,
	SeekTime:            10 * time.Millisecond,
	ReadBytesPerSecond:  100 * Mebibyte,
	WriteBytesPerSecond: 100 * Mebibyte,
	// Default to 4096 times faster than writing, since ext4 block sizes are
	// 4 KiB.
	AllocateBytesPerSecond: 4096 * 100 * Mebibyte,
	RequestReorderMaxDelay: 100 * time.Microsecond,
	FsyncStrategy:          WriteBackCachedFsync,
	WriteStrategy:          FastWrite,
	MetadataOpTime:         10 * time.Millisecond,
}
