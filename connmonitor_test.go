package connmonitor

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/uplo-tech/fastrand"
)

// testConn is a wrapper of a net.Conn interface to be used for testing
type testConn struct {
	net.Conn
}

func (tc testConn) Read(b []byte) (int, error) {
	return fastrand.Intn(len(b)), nil
}
func (tc testConn) Write(b []byte) (int, error) {
	return fastrand.Intn(len(b)), nil
}

// TestSingleMonitoredWriteRead tests a single monitored write and read
// operation to ensure that the underlying Monitor's counts are updated
// correctly.
func TestSingleMonitoredWriteRead(t *testing.T) {
	// Create monitor
	m := NewMonitor()

	// Wrap conn into a monitored counter.
	conn := testConn{}
	mc := NewMonitoredConn(conn, m)

	// Create 1mb to write.
	data := fastrand.Bytes(1000)

	// Write data
	n, err := mc.Write(data)
	if err != nil {
		t.Error("Failed to write data", err)
	}
	// Check the counter was incremented
	writeBytes := atomic.LoadUint64(&m.atomicWriteBytes)
	if writeBytes != uint64(n) {
		t.Errorf("Expected writeBytes to be %v but was %v", n, writeBytes)
	}

	// Read data back from file.
	n, err = mc.Read(data)
	if err != nil {
		t.Error("Failed to read data", err)
	}
	// Check the counter was incremented
	readBytes := atomic.LoadUint64(&m.atomicReadBytes)
	if readBytes != uint64(n) {
		t.Errorf("Expected readBytes to be %v but was %v", n, readBytes)
	}
}

// TestMultipleMonitoredWriteRead tests multiple parallel monitored conn write
// and read operations to ensure that the common Monitor is updating the counts
// correctly.
func TestMultipleMonitoredWriteRead(t *testing.T) {
	// Create monitor
	m := NewMonitor()
	bytesToWrite := int(1000)

	// totalWrite and totalRead are to track how much data was actually written
	// and read by the Write and Read calls
	var atomicTotalWrite, atomicTotalRead uint64

	// f creates a monitored conn, writes some data and reads it
	f := func() {
		// Wrap conn into a monitored conn.
		conn := testConn{}
		mc := NewMonitoredConn(conn, m)

		// Create 1mb to write.
		data := fastrand.Bytes(bytesToWrite)

		// Write data.
		n, err := mc.Write(data)
		if err != nil {
			t.Error("Failed to write data", err)
		}
		atomic.AddUint64(&atomicTotalWrite, uint64(n))

		// Read data back from file.
		n, err = mc.Read(data)
		if err != nil {
			t.Error("Failed to read data", err)
		}
		atomic.AddUint64(&atomicTotalRead, uint64(n))
	}
	// Start a few threads and wait for them to finish.
	var wg sync.WaitGroup
	numThreads := 10
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			f()
			wg.Done()
		}()
	}
	wg.Wait()

	// Check the counters was incremented
	if m.atomicWriteBytes != atomicTotalWrite {
		t.Errorf("Expected writeBytes to be %v but was %v", atomicTotalWrite, m.atomicWriteBytes)
	}
	if m.atomicReadBytes != atomicTotalRead {
		t.Errorf("Expected readBytes to be %v but was %v", atomicTotalRead, m.atomicReadBytes)
	}
}
