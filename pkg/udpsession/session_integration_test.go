// Copyright (C) 2021  mieru authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package udpsession_test

import (
	"bytes"
	"context"
	"fmt"
	mrand "math/rand"
	"sync"
	"testing"
	"time"

	"github.com/enfein/mieru/pkg/appctl/appctlpb"
	"github.com/enfein/mieru/pkg/cipher"
	"github.com/enfein/mieru/pkg/rng"
	"github.com/enfein/mieru/pkg/testtool"
	"github.com/enfein/mieru/pkg/udpsession"
)

var users = map[string]*appctlpb.User{
	"dengxiaoping": {
		Name:     "dengxiaoping",
		Password: "19890604",
	},
	"jiangzemin": {
		Name:     "jiangzemin",
		Password: "20001027",
	},
	"hujintao": {
		Name:     "hujintao",
		Password: "20080512",
	},
	"xijinping": {
		Name:     "xijinping",
		Password: "20200630",
	},
}

func runClient(t *testing.T, laddr, serverAddr string, username, password []byte, writeSize, readSize int) error {
	hashedPassword := cipher.HashPassword(password, username)
	block, err := cipher.BlockCipherFromPassword(hashedPassword, true)
	if err != nil {
		return fmt.Errorf("cipher.BlockCipherFromPassword() failed: %w", err)
	}
	sess, err := udpsession.DialWithOptions(context.Background(), "udp", laddr, serverAddr, block)
	if err != nil {
		return fmt.Errorf("udpsession.DialWithOptions() failed: %w", err)
	}
	defer sess.Close()
	t.Logf("[%s] client is running on %v", time.Now().Format(testtool.TimeLayout), laddr)

	for i := 0; i < 50; i++ {
		sleepMillis := 50 + mrand.Intn(50)
		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)

		resp := make([]byte, 0, writeSize)
		respBuf := make([]byte, readSize)
		data := testtool.TestHelperGenRot13Input(writeSize)

		// Send data to server.
		if _, err = sess.Write(data); err != nil {
			return fmt.Errorf("Write() failed: %w", err)
		}
		sess.SetReadDeadline(time.Now().Add(1 * time.Second))

		// Get and verify server response.
		totalSize := 0
		for totalSize < writeSize {
			size, err := sess.Read(respBuf)
			if err != nil {
				return fmt.Errorf("Read() failed: %w", err)
			}
			resp = append(resp, respBuf[:size]...)
			totalSize += size
		}
		if totalSize != writeSize {
			return fmt.Errorf("read %d bytes, want %d bytes", totalSize, writeSize)
		}
		revert, err := testtool.TestHelperRot13(resp[:totalSize])
		if err != nil {
			return fmt.Errorf("testtool.TestHelperRot13() failed: %w", err)
		}
		if !bytes.Equal(data, revert) {
			return fmt.Errorf("verification failed")
		}
	}
	return nil
}

// TestKCPSessionsIPv4 creates one listener and two clients. Each client sends
// some data (in format [A-Za-z]+) to the listener. The listener returns the
// ROT13 (rotate by 13 places) of the data back to the client.
func TestKCPSessionsIPv4(t *testing.T) {
	rng.InitSeed()
	party, err := udpsession.ListenWithOptions("127.0.0.1:12315", users)
	if err != nil {
		t.Fatalf("udpsession.ListenWithOptions() failed: %v", err)
	}

	go func() {
		for {
			s, err := party.Accept()
			if err != nil {
				return
			} else {
				t.Logf("[%s] accepting new connection from %v", time.Now().Format(testtool.TimeLayout), s.RemoteAddr())
				go func() {
					if err = testtool.TestHelperServeConn(s); err != nil {
						return
					}
				}()
			}
		}
	}()
	time.Sleep(1 * time.Second)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		if err := runClient(t, "127.0.0.1:12316", "127.0.0.1:12315", []byte("jiangzemin"), []byte("20001027"), 1024*64, 9000); err != nil {
			t.Errorf("[%s] jiangzemin failed: %v", time.Now().Format(testtool.TimeLayout), err)
		}
		wg.Done()
	}()
	go func() {
		if err := runClient(t, "127.0.0.1:12317", "127.0.0.1:12315", []byte("xijinping"), []byte("20200630"), 9000, 1024*64); err != nil {
			t.Errorf("[%s] xijinping failed: %v", time.Now().Format(testtool.TimeLayout), err)
		}
		wg.Done()
	}()
	wg.Wait()

	party.Close()
	time.Sleep(1 * time.Second) // Wait for resources to be released.
}

// TestKCPSessionsIPv6 is similar to TestKCPSessionsIPv4 but running in IPv6.
func TestKCPSessionsIPv6(t *testing.T) {
	rng.InitSeed()
	party, err := udpsession.ListenWithOptions("[::1]:12318", users)
	if err != nil {
		t.Fatalf("udpsession.ListenWithOptions() failed: %v", err)
	}

	go func() {
		for {
			s, err := party.Accept()
			if err != nil {
				return
			} else {
				t.Logf("[%s] accepting new connection from %v", time.Now().Format(testtool.TimeLayout), s.RemoteAddr())
				go func() {
					if err = testtool.TestHelperServeConn(s); err != nil {
						return
					}
				}()
			}
		}
	}()
	time.Sleep(1 * time.Second)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		if err := runClient(t, "[::1]:12319", "[::1]:12318", []byte("jiangzemin"), []byte("20001027"), 1024*64, 9000); err != nil {
			t.Errorf("[%s] jiangzemin failed: %v", time.Now().Format(testtool.TimeLayout), err)
		}
		wg.Done()
	}()
	go func() {
		if err := runClient(t, "[::1]:12320", "[::1]:12318", []byte("xijinping"), []byte("20200630"), 9000, 1024*64); err != nil {
			t.Errorf("[%s] xijinping failed: %v", time.Now().Format(testtool.TimeLayout), err)
		}
		wg.Done()
	}()
	wg.Wait()

	party.Close()
	time.Sleep(1 * time.Second) // Wait for resources to be released.
}
